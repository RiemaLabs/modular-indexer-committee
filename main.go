package main

import (
	"encoding/json"
	"log"
	"os"
	"runtime/debug"
	"time"

	"github.com/ethereum/go-verkle"
)

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

	// Fetch the latest block height.
	latestHeight, err := getter.GetLatestBlockHeight()

	if err != nil {
		log.Fatalf("Failed to get the latest block height: %v", err)
	}

	// New queue from the scratch.
	// TODO: Store the stateRoot to the local disk and reload them.
	currentLatestHeight := BRC20StartHeight - 1
	stateRoot := verkle.New()

	if latestHeight-BitcoinConfirmations > currentLatestHeight {
		s, err := FastCatchup(getter, stateRoot, currentLatestHeight, latestHeight-BitcoinConfirmations)
		stateRoot = s
		if err != nil {
			log.Fatalf("Failed to catchup the latest block height: %v", err)
		}
	} else if latestHeight-BitcoinConfirmations == currentLatestHeight {
		// stateRoot is located at catchupHeight.
	} else if latestHeight-BitcoinConfirmations < currentLatestHeight {
		log.Fatalf("Stored stateRoot is beyond the Bitcoin confirmations.")
	}

	catchupHeight := latestHeight - BitcoinConfirmations + 1
	queue, err := NewQueues(getter, stateRoot, false, catchupHeight)

	if err != nil {
		log.Fatalf("Failed to initial queues: %v", err)
	}

	// Provide service
	var history = make(map[uint]map[string]bool)

	for {
		originalLatestHeight := queue.LastestHeight()
		latestHeight, err := getter.GetLatestBlockHeight()
		if err != nil {
			log.Fatalf("Failed to get the latest block height: %v", err)
		}

		if originalLatestHeight < latestHeight {
			queue.Lock()
			err := queue.Update(getter, queue.LastestStateRoot(), originalLatestHeight, latestHeight)
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
			err := queue.Update(getter, queue.StateRoot(reorgHeight), reorgHeight, latestHeight)
			if err != nil {
				log.Fatalf("Failed to update the queue: %v", err)
			}
		}
		queue.Unlock()

		if arguments.EnableService {
			for i := 0; i <= len(queue.states)-1; i++ {
				go Upload(history, GlobalConfig, queue.states[i].Checkpoint(GlobalConfig))
			}
		}

		time.Sleep(60 * time.Second)
	}
}
