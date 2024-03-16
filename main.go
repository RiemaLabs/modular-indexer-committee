package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime/debug"
	"time"

	"github.com/RiemaLabs/indexer-committee/checkpoint"
	"github.com/RiemaLabs/indexer-committee/ord"
	"github.com/RiemaLabs/indexer-committee/ord/getter"
	"github.com/RiemaLabs/indexer-committee/storage"
)

func catchupStage(getter getter.OrdGetter, arguments *RuntimeArguments, initHeight uint) (*ord.StateQueue, error) {
	// Fetch the latest block height.
	latestHeight, err := getter.GetLatestBlockHeight()
	if err != nil {
		return nil, err
	}
	catchupHeight := latestHeight - ord.BitcoinConfirmations

	state := storage.LoadState(arguments.EnableStateRootCache, initHeight)
	curHeight := state.Height

	// Start to catch-up

	if catchupHeight > curHeight {
		// TODO: Refine the catchup performance by batching query.
		log.Printf("Fast catchup to the lateset block height! From %d to %d \n", curHeight, catchupHeight)

		for i := curHeight; i <= catchupHeight; i++ {
			ordTransfer, err := getter.GetOrdTransfers(i)
			if err != nil {
				return nil, err
			}
			state = ord.Exec(state, ordTransfer)
			if i%1000 == 0 {
				log.Printf("Blocks: %d / %d \n", i, catchupHeight)
				if arguments.EnableStateRootCache {
					err := storage.StoreState(state, state.Height-2000)
					if err != nil {
						log.Printf("Failed to store the cache at height: %d", i)
					}
				}
			}
			state.Height += 1
		}
	} else if catchupHeight == curHeight {
		// stateRoot is located at catchupHeight.
	} else if catchupHeight < curHeight {
		return nil, errors.New("the stored stateRoot is too advanced to handle reorg situations")
	}
	if arguments.EnableStateRootCache {
		err := storage.StoreState(state, state.Height-2000)
		if err != nil {
			log.Printf("Failed to store the cache at height: %d", catchupHeight)
		}
	}
	return ord.NewQueues(getter, state, false, catchupHeight+1)
}

func serviceStage(getter getter.OrdGetter, arguments *RuntimeArguments, queue *ord.StateQueue) {
	// Provide service
	var history = make(map[uint]map[string]bool)

	for {
		curHeight := queue.LastestHeight()
		latestHeight, err := getter.GetLatestBlockHeight()
		if err != nil {
			log.Fatalf("Failed to get the latest block height: %v", err)
		}

		if curHeight < latestHeight {
			queue.Lock()
			err := queue.Update(getter, queue.State(curHeight), latestHeight)
			queue.Unlock()
			if err != nil {
				log.Fatalf("Failed to update the queue: %v", err)
			}
		}

		queue.Lock()
		reorgHeight, err := queue.CheckForReorg(getter)

		if err != nil {
			log.Fatalf("Failed to check the reorganization: %v", err)
		}

		if reorgHeight != 0 {
			err := queue.Update(getter, queue.State(reorgHeight), latestHeight)
			if err != nil {
				log.Fatalf("Failed to update the queue: %v", err)
			}
		}
		queue.Unlock()

		if arguments.EnableService {
			indexerID := checkpoint.IndexerIdentification{
				URL:          GlobalConfig.Service.URL,
				Name:         GlobalConfig.Service.Name,
				Version:      Version,
				MetaProtocol: GlobalConfig.Service.MetaProtocol,
			}
			for i := 0; i <= len(queue.States)-1; i++ {
				c := checkpoint.NewCheckpoint(indexerID, queue.States[i])
				go checkpoint.UploadCheckpoint(history, indexerID, c)
			}
		}

		time.Sleep(60 * time.Second)
	}
}

func main() {

	// 启动 HTTP 服务器，提供 pprof 分析接口
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()

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

	// Use OPI database as the getter.
	getter, err := getter.NewOPIBitcoinGetter(getter.DatabaseConfig(GlobalConfig.Database))

	if err != nil {
		log.Fatalf("Failed to initial getter from opi database: %v", err)
	}

	queue, err := catchupStage(getter, arguments, ord.BRC20StartHeight-1)

	if err != nil {
		log.Fatalf("Failed to catchup the latest state: %v", err)
	}

	serviceStage(getter, arguments, queue)
}
