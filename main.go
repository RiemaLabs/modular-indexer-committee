package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"runtime/debug"
	"time"

	"github.com/ethereum/go-verkle"
)

func catchupStage(getter BitcoinGetter, arguments *RuntimeArguments) (*StateQueue, error) {
	// Fetch the latest block height.
	latestHeight, err := getter.GetLatestBlockHeight()

	if err != nil {
		return nil, err
	}

	// New queue from the scratch.

	cachePath := ".cache"
	curHeight := BRC20StartHeight - 1
	stateRoot := verkle.New()
	state := State{
		root:   stateRoot,
		kv:     make(KeyValueMap),
		height: curHeight,
		hash:   "",
	}

	if arguments.EnableStateRootCache {
		storedState, err := DeserializeLatestState(cachePath)
		if err != nil {
			log.Printf("Warning for loading stateRoot %v\n", err)
		}
		if storedState != nil {
			state = *storedState
			curHeight = state.height
		}
	}

	catchupHeight := latestHeight - BitcoinConfirmations

	if catchupHeight > curHeight {
		// TODO: Refine the catchup performance by batching query.
		log.Printf("Fast catchup to the lateset block height! From %d to %d \n", curHeight, catchupHeight)

		for i := curHeight; i <= catchupHeight; i++ {
			ordTransfer, err := getter.GetOrdTransfers(i)
			if err != nil {
				return nil, err
			}
			state = processOrdTransfer(state, ordTransfer)
			if i%1000 == 0 {
				log.Printf("Blocks: %d / %d \n", i, catchupHeight)
				if arguments.EnableStateRootCache {
					err := state.SerializeToFile(cachePath)
					if err != nil {
						log.Printf("Warning for saving stateRoot %v\n", err)
					}
				}
			}
			state.height += 1
		}
	} else if catchupHeight == curHeight {
		// stateRoot is located at catchupHeight.
	} else if catchupHeight < curHeight {
		return nil, errors.New("the stored stateRoot is too advanced to handle reorg situations")
	}
	return NewQueues(getter, state, false, catchupHeight+1)
}

func serviceStage(getter BitcoinGetter, arguments *RuntimeArguments, queue *StateQueue) {
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
			for i := 0; i <= len(queue.states)-1; i++ {
				go UploadCheckpoint(history, GlobalConfig, queue.states[i].Checkpoint(GlobalConfig))
			}
		}

		time.Sleep(60 * time.Second)
	}
}

func main() {

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
	getter, err := NewOPIBitcoinGetter(GlobalConfig)

	if err != nil {
		log.Fatalf("Failed to initial getter from opi database: %v", err)
	}

	queue, err := catchupStage(getter, arguments)

	if err != nil {
		log.Fatalf("Failed to catchup the latest state: %v", err)
	}

	serviceStage(getter, arguments, queue)
}
