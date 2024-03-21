package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	_ "net/http/pprof"
	"os"
	"runtime/debug"
	"time"

	"github.com/RiemaLabs/indexer-committee/ord"
	"github.com/RiemaLabs/indexer-committee/ord/getter"
	"github.com/RiemaLabs/indexer-committee/ord/stateless"
)

func catchupStage(ordGetter getter.OrdGetter, arguments *RuntimeArguments, initHeight uint, latestHeight uint) (*stateless.Queue, error) {
	// Fetch the latest block height.
	header := stateless.LoadHeader(arguments.EnableStateRootCache, initHeight)
	curHeight := header.Height

	log.Printf("Fast catchup to the lateset block height! From %d to %d \n", curHeight, latestHeight)

	catchupHeight := latestHeight - ord.BitcoinConfirmations + 1

	// Start to catch-up
	// TODO: Medium. Refine the catchup performance by batching query.
	if catchupHeight > curHeight {
		for i := curHeight + 1; i <= catchupHeight; i++ {
			ordTransfer, err := ordGetter.GetOrdTransfers(i)
			if err != nil {
				return nil, err
			}
			stateless.Exec(&header, ordTransfer, i)
			// header.Height ++
			header.Paging(ordGetter, false, stateless.NodeResolveFn)
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
	} else if catchupHeight == curHeight {
		// stateRoot is located at catchupHeight.
	} else if catchupHeight < curHeight {
		return nil, errors.New("the stored stateRoot is too advanced to handle reorg situations")
	}

	// header.Height == catchupHeight

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

	queue, err := stateless.NewQueues(ordGetter, &header, true, catchupHeight+1)
	if err != nil {
		return nil, err
	}
	if queue.LatestHeight() != latestHeight {
		return nil, fmt.Errorf("mismatched state height: %d and catchup height: %d", queue.LatestHeight(), latestHeight)
	}
	return queue, nil
}

func serviceStage(ordGetter getter.OrdGetter, arguments *RuntimeArguments, queue *stateless.Queue, interval time.Duration) {
	// TODO: Urgent. Provide service.
	// var history = make(map[uint]map[string]bool)

	for {
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

		if arguments.EnableService {
			// indexerID := checkpoint.IndexerIdentification{
			// 	URL:          GlobalConfig.Service.URL,
			// 	Name:         GlobalConfig.Service.Name,
			// 	Version:      Version,
			// 	MetaProtocol: GlobalConfig.Service.MetaProtocol,
			// }
			// for i := 0; i <= len(queue.States)-1; i++ {
			// 	c := checkpoint.NewCheckpoint(indexerID, queue.States[i])
			// 	go checkpoint.UploadCheckpoint(history, indexerID, c)
			// }
		}

		time.Sleep(interval)
	}
}

func main() {
	// printCache()
	// return
	arguments := NewRuntimeArguments()
	rootCmd := arguments.MakeCmd()
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Failed to parse the arguments: %v", err)
	}

	// Get the version as a stamp for the checkpoint.
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		log.Fatalf("Failed to obtain build information.")
	}
	Version = bi.Main.Version

	// Get the configuration.
	configFile, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	err = json.Unmarshal(configFile, &GlobalConfig)
	if err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	// Use OPI database as the ordGetter.
	ordGetter, err := getter.NewOPIBitcoinGetter(getter.DatabaseConfig(GlobalConfig.Database))

	if err != nil {
		log.Fatalf("Failed to initial getter from opi database: %v", err)
	}

	latestHeight, err := ordGetter.GetLatestBlockHeight()
	if err != nil {
		log.Fatalf("Failed to get the latest block height: %v", err)
	}

	queue, err := catchupStage(ordGetter, arguments, stateless.BRC20StartHeight-1, latestHeight-ord.BitcoinConfirmations)

	if err != nil {
		log.Fatalf("Failed to catchup the latest state: %v", err)
	}

	serviceStage(ordGetter, arguments, queue, 60*time.Second)
}
