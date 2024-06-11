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
	"github.com/RiemaLabs/modular-indexer-committee/internal/metrics"
	"github.com/RiemaLabs/modular-indexer-committee/ord"
	"github.com/RiemaLabs/modular-indexer-committee/ord/getter"
	"github.com/RiemaLabs/modular-indexer-committee/ord/stateless"
)

var (
	version = "latest"
	gitHash = "unknown"
)

func CatchupStage(okxGetter getter.OrdGetter, arguments *RuntimeArguments, initHeight uint, latestHeight uint) (*stateless.Queue, error) {
	metrics.Stage.Set(metrics.StageCatchup)

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
				_ = stateless.StoreHeader(header, header.Height-2000)
				os.Exit(0)
			default:
				ordTransfer, err := okxGetter.GetOrdTransfers(i)
				if err != nil {
					return nil, err
				}
				header.Lock()
				stateless.Exec(header, ordTransfer, i)
				_ = header.Paging(okxGetter, false, stateless.NodeResolveFn)
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
	ots, err := okxGetter.GetOrdTransfers(catchupHeight)
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

	queue, err := stateless.NewQueues(okxGetter, header, true, catchupHeight+1)
	if err != nil {
		return nil, err
	}
	if queue.LatestHeight() != latestHeight {
		return nil, fmt.Errorf("mismatched state height: %d and catchup height: %d", queue.LatestHeight(), latestHeight)
	}
	return queue, nil
}

func ServiceStage(ordGetter getter.OrdGetter, arguments *RuntimeArguments, queue *stateless.Queue, interval time.Duration) {
	metrics.Stage.Set(metrics.StageServing)

	// Create a channel to listen for SIGINT (Ctrl+C) signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	var history = make(map[string]checkpoint.UploadRecord)

	if arguments.EnableService {
		if arguments.CommitteeIndexerURL != "" {
			log.Printf("Providing API service at: %s", arguments.CommitteeIndexerURL)
		} else {
			log.Printf("Providing API service at: %s", GlobalConfig.Service.URL)
		}
		go apis.StartService(queue, arguments.EnableCommittee, arguments.EnableTest, arguments.EnablePprof)
	}

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
				metrics.Stage.Set(metrics.StageUpdating)
				err := queue.Update(ordGetter, latestHeight)
				if err != nil {
					log.Fatalf("Failed to update the queue: %v", err)
				}
				metrics.Stage.Set(metrics.StageServing)
			}

			reorgHeight, err := queue.CheckForReorg(ordGetter)

			if err != nil {
				log.Fatalf("Failed to check the reorganization: %v", err)
			}

			if reorgHeight != 0 {
				metrics.Stage.Set(metrics.StageReorg)
				err := queue.Recovery(ordGetter, reorgHeight)
				if err != nil {
					log.Fatalf("Failed to update the queue: %v", err)
				}
				metrics.Stage.Set(metrics.StageServing)
			}

			if arguments.EnableCommittee {
				latestHistory := stateless.DiffState{
					Height:       queue.Header.Height,
					Hash:         queue.Header.Hash,
					VerkleCommit: queue.Header.Root.Commit().Bytes(),
					Access:       stateless.AccessList{},
				}
				hs := make([]*stateless.DiffState, 0)
				for _, i := range queue.History {
					hs = append(hs, &i)
				}
				hs = append(hs, &latestHistory)
				for _, i := range hs {
					key := fmt.Sprintf("%d", i.Height) + i.Hash
					if curRecord, found := history[key]; !(found && curRecord.Success) {
						committeeIndexerName := GlobalConfig.Service.Name
						if arguments.CommitteeIndexerName != "" {
							committeeIndexerName = arguments.CommitteeIndexerName
						}
						serviceURL := GlobalConfig.Service.URL
						if arguments.CommitteeIndexerURL != "" {
							serviceURL = arguments.CommitteeIndexerURL
						}
						metaProtocol := GlobalConfig.Service.MetaProtocol
						if arguments.ProtocolName != "" {
							metaProtocol = arguments.ProtocolName
						}
						indexerID := checkpoint.IndexerIdentification{
							URL:          serviceURL,
							Name:         committeeIndexerName,
							Version:      version,
							MetaProtocol: metaProtocol,
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
								log.Printf("Unable to upload the checkpoint by S3 due to: %v", err)
							} else {
								log.Printf("Succeed to upload the checkpoint by S3 at height: %s\n", c.Height)
							}
						} else if GlobalConfig.Report.Method == "DA" {
							log.Printf("Uploading the checkpoint by DA at height: %s\n", c.Height)
							dacfg := GlobalConfig.Report.Da
							err = checkpoint.UploadCheckpointByDA(&c,
								dacfg.PrivateKey, dacfg.GasCoupon, dacfg.NamespaceID, dacfg.Network, timeout)
							if err != nil {
								log.Printf("Unable to upload the checkpoint by DA due to: %v", err)
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
			if !arguments.EnableTest {
				log.Printf("Listening for new Bitcoin block, current height: %d\n", latestHeight)
			}
			time.Sleep(interval)
		}
	}
}

func Execution(arguments *RuntimeArguments) {
	go metrics.ListenAndServe(arguments.MetricAddr)
	metrics.Version.WithLabelValues(version).Set(1)
	metrics.Stage.Set(metrics.StageInitializing)

	// Get the configuration.
	configFile, err := os.ReadFile(arguments.ConfigFilePath)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	err = json.Unmarshal(configFile, &GlobalConfig)
	if err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

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
			nid, err := checkpoint.CreateNamespace(GlobalConfig.Report.Da.PrivateKey, GlobalConfig.Report.Da.GasCoupon, namespaceName, GlobalConfig.Report.Da.Network)
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

	// Use OKX database as the ordGetter.
	gd := getter.DatabaseConfig(GlobalConfig.Database)
	var okxGetter getter.OrdGetter
	if arguments.EnableTest {
		okxGetter, err = getter.NewOKXBRC20GetterTest(&gd, arguments.TestBlockHeightLimit, arguments.TestBlockHeightLimit)
	} else {
		okxGetter, err = getter.NewOKXBRC20Getter(&gd)
	}
	if err != nil {
		log.Fatalf("Failed to initial getter from okx database: %v", err)
	}

	latestHeight, err := okxGetter.GetLatestBlockHeight()
	if err != nil {
		log.Fatalf("Failed to get the latest block height: %v", err)
	}

	queue, err := CatchupStage(okxGetter, arguments, stateless.BRC20StartHeight-1, latestHeight)

	if err != nil {
		log.Fatalf("Failed to catchup the latest state: %v", err)
	}

	ServiceStage(okxGetter, arguments, queue, 60*time.Second)
}

func main() {
	arguments := NewRuntimeArguments()
	rootCmd := arguments.MakeCmd()
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Failed to execute: %v", err)
	}
}
