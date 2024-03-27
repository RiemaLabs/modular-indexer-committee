package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/RiemaLabs/modular-indexer-committee/apis"
	"github.com/RiemaLabs/modular-indexer-committee/checkpoint"
	"github.com/RiemaLabs/modular-indexer-committee/ord"
	"github.com/RiemaLabs/modular-indexer-committee/ord/getter"
	"github.com/RiemaLabs/modular-indexer-committee/ord/stateless"
)

func catchupStage(ordGetter getter.OrdGetter, arguments *RuntimeArguments, initHeight uint, latestHeight uint) (*stateless.Queue, error) {
	// Fetch the latest block height.
	header := stateless.LoadHeader(arguments.EnableStateRootCache, initHeight)
	curHeight := header.Height

	log.Printf("Fast catchup to the lateset block height! From %d to %d \n", curHeight, latestHeight)

	catchupHeight := latestHeight - ord.BitcoinConfirmations

	// Create a channel to listen for SIGINT (Ctrl+C) signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	// Start to catch-up
	// TODO: Medium. Refine the catchup performance by batching query.
	if catchupHeight > curHeight {
		for i := curHeight + 1; i <= catchupHeight; i++ {
			select {
			case <-sigChan:
				// SIGINT received, stop the catch-up process
				log.Printf("Saving cache file. Please don't force exit.")
				stateless.StoreHeader(header, header.Height-2000)
				os.Exit(0)
			default:
				ordTransfer, err := ordGetter.GetOrdTransfers(i)
				if err != nil {
					return nil, err
				}
				header.Lock()
				stateless.Exec(header, ordTransfer, i)
				// header.Height ++
				header.Paging(ordGetter, false, stateless.NodeResolveFn)
				header.Unlock()
				if i%1000 == 0 {
					log.Printf("Blocks: %d / %d \n", i, catchupHeight)
					if arguments.EnableStateRootCache {
						err := stateless.StoreHeader(header, header.Height-2000)
						if err != nil {
							log.Printf("Failed to store the cache at height: %d", i)
						}
					}
				}
			}
		}
	} else if catchupHeight == curHeight {
		// stateRoot is located at catchupHeight.
	} else if catchupHeight < curHeight {
		return nil, errors.New("the stored stateRoot is too advanced to handle reorg situations")
	}

	// Currently, header.Height equals to catchupHeight.

	ots, err := ordGetter.GetOrdTransfers(catchupHeight)
	if err != nil {
		return nil, err
	}
	header.OrdTrans = ots

	if arguments.EnableStateRootCache {
		err := stateless.StoreHeader(header, header.Height-2000)
		if err != nil {
			log.Printf("Failed to store the cache at height: %d", header.Height)
		}
	}

	queue, err := stateless.NewQueues(ordGetter, header, true, catchupHeight+1)
	if err != nil {
		return nil, err
	}
	if queue.LatestHeight() != latestHeight {
		return nil, fmt.Errorf("mismatched state height: %d and catchup height: %d", queue.LatestHeight(), latestHeight)
	}
	return queue, nil
}

func serviceStage(ordGetter getter.OrdGetter, arguments *RuntimeArguments, queue *stateless.Queue, interval time.Duration) {
	// Create a channel to listen for SIGINT (Ctrl+C) signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	var history = make(map[string]checkpoint.UploadRecord)

	for {
		select {
		case <-sigChan:
			// TODO: High. Save the latest state is unsound if reorg happened.
			// log.Printf("Saving cache file. Please don't force exit.")
			// stateless.StoreHeader(queue.Header, queue.Header.Height-2000)
			os.Exit(0)
		default:
			curHeight := queue.LatestHeight()
			latestHeight, err := ordGetter.GetLatestBlockHeight()
			if err != nil {
				log.Fatalf("Failed to get the latest block height: %v", err)
			}

			if curHeight < latestHeight {
				queue.Lock()
				err := queue.Update(ordGetter, latestHeight)
				queue.Unlock()
				if err != nil {
					log.Fatalf("Failed to update the queue: %v", err)
				}
			}

			queue.Lock()
			reorgHeight, err := queue.CheckForReorg(ordGetter)

			if err != nil {
				log.Fatalf("Failed to check the reorganization: %v", err)
			}

			if reorgHeight != 0 {
				err := queue.Recovery(ordGetter, reorgHeight)
				if err != nil {
					log.Fatalf("Failed to update the queue: %v", err)
				}
			}
			queue.Unlock()

			if arguments.EnableCommittee {
				latestHistory := stateless.DiffState{
					Height:       queue.Header.Height,
					Hash:         queue.Header.Hash,
					VerkleCommit: queue.Header.Root.Commit().Bytes(),
					Diff:         stateless.DiffList{},
				}
				hs := make([]*stateless.DiffState, 0)
				for _, i := range queue.History {
					hs = append(hs, &i)
				}
				hs = append(hs, &latestHistory)
				for _, i := range hs {
					key := fmt.Sprintf("%d", i.Height) + i.Hash
					if curRecord, found := history[key]; !(found && curRecord.Success) {
						indexerID := checkpoint.IndexerIdentification{
							URL:          GlobalConfig.Service.URL,
							Name:         GlobalConfig.Service.Name,
							Version:      Version,
							MetaProtocol: GlobalConfig.Service.MetaProtocol,
						}
						commitment := base64.StdEncoding.EncodeToString(i.VerkleCommit[:])
						c := checkpoint.NewCheckpoint(&indexerID, i.Height, i.Hash, commitment)
						timeout := time.Duration(GlobalConfig.Report.Timeout) * time.Millisecond
						if GlobalConfig.Report.Method == "S3" {
							log.Printf("Uploading the checkpoint by S3 at height: %s\n", c.Height)
							s3cfg := GlobalConfig.Report.S3
							err = checkpoint.UploadCheckpointByS3(&c,
								s3cfg.AccessKey, s3cfg.SecretKey, s3cfg.Region, s3cfg.Bucket, timeout)
							if err != nil {
								log.Fatalf("Unable to upload the checkpoint by S3 due to: %v", err)
							} else {
								log.Printf("Succeed to upload the checkpoint by S3 at height: %s\n", c.Height)
							}
						} else if GlobalConfig.Report.Method == "DA" {
							log.Printf("Uploading the checkpoint by DA at height: %s\n", c.Height)
							dacfg := GlobalConfig.Report.Da
							err = checkpoint.UploadCheckpointByDA(&c,
								dacfg.PrivateKey, dacfg.GasCode, dacfg.NamespaceID, dacfg.Network, timeout)
							if err != nil {
								log.Fatalf("Unable to upload the checkpoint by DA due to: %v", err)
							} else {
								log.Printf("Succeed to upload the checkpoint by DA at height: %s\n", c.Height)
							}
						}
						history[key] = checkpoint.UploadRecord{
							Success: true,
						}
					}
				}
			}

			if arguments.EnableService {
				log.Printf("Providing API service at: %s", GlobalConfig.Service.URL)
				go apis.StartService(queue, arguments.EnableCommittee, arguments.EnableTest)
			}

			log.Printf("Listening for new Bitcoin block, current height: %d\n", latestHeight)
			time.Sleep(interval)
		}
	}
}

func Execution(arguments *RuntimeArguments) {
	// Get the configuration.
	configFile, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	err = json.Unmarshal(configFile, &GlobalConfig)
	if err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	Version = GlobalConfig.Service.Version

	if GlobalConfig.Report.Method == "DA" && arguments.EnableCommittee {
		if !checkpoint.IsValidNamespaceID(GlobalConfig.Report.Da.NamespaceID) {
			log.Printf("Got invalid Namespace ID from the config.json. Initializing a new namespace.")
			scanner := bufio.NewScanner(os.Stdin)
			namespaceName := ""
			for {
				fmt.Print("Please enter the namespace name: ")
				if scanner.Scan() {
					namespaceName = scanner.Text()
					if strings.TrimSpace(namespaceName) == "" {
						fmt.Print("Namespace name couldn't be empty!")
					} else {
						break
					}
				}
			}
			nid, err := checkpoint.CreateNamespace(GlobalConfig.Report.Da.PrivateKey, GlobalConfig.Report.Da.GasCode, namespaceName, GlobalConfig.Report.Da.Network)
			if err != nil {
				log.Fatalf("Failed to create namespace due to %v", err)
			}
			GlobalConfig.Report.Da.NamespaceID = nid
			bytes, err := json.Marshal(GlobalConfig)
			if err != nil {
				log.Fatalf("Failed to save namespace ID to local file due to %v", err)
			}
			err = os.WriteFile("config.json", bytes, 0644)
			if err != nil {
				log.Fatalf("Failed to save namespace ID to local file due to %v", err)
			}
			fmt.Printf("Succeed to create namespace, ID: %s!", nid)
		}
	}

	// Use OPI database as the ordGetter.
	gd := getter.DatabaseConfig(GlobalConfig.Database)
	var ordGetter getter.OrdGetter
	if arguments.EnableTest {
		ordGetter, err = getter.NewOPIOrdGetterTest(&gd, arguments.TestBlockHeightLimit)
	} else {
		ordGetter, err = getter.NewOPIOrdGetter(&gd)
	}
	if err != nil {
		log.Fatalf("Failed to initial getter from opi database: %v", err)
	}

	latestHeight, err := ordGetter.GetLatestBlockHeight()
	if err != nil {
		log.Fatalf("Failed to get the latest block height: %v", err)
	}

	queue, err := catchupStage(ordGetter, arguments, stateless.BRC20StartHeight-1, latestHeight)

	if err != nil {
		log.Fatalf("Failed to catchup the latest state: %v", err)
	}

	serviceStage(ordGetter, arguments, queue, 60*time.Second)
}

func main() {
	arguments := NewRuntimeArguments()
	rootCmd := arguments.MakeCmd()
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Failed to execute: %v", err)
	}
}
