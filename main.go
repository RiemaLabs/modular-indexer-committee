package main

import (
	"encoding/json"
	"log"
	"os"
	"runtime/debug"

	"github.com/ethereum/go-verkle"
)

func main() {

	arguments := NewRuntimeArguments()
	rootCmd := arguments.MakeCmd()
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Failed to parse the arguments", err)
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

	// Connect to the OPI database.
	db, err := ConnectOPIDatabase(GlobalConfig)

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Recover the latest state root.
	stateRoot := verkle.New()
	verkles := InitVerkleSlotsFromScratch(BitcoinConfirmations, BRC20StartHeight)
	verkles.initCommittee(db, stateRoot, latestHeight)
	go verkles.updateCommittee(db)

	// // Fetch the latest block height.
	// latestHeight, err := FetchBlockHeight(GlobalConfig)

	// if err != nil {
	// 	log.Fatalf("Failed to read from the rpc: %v", err)
	// }

	// TODO: Read local cache.

	// Fetcher Goroutine

	//
}
