package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/RiemaLabs/modular-indexer-committee/ord/getter"
)

func loadMain(hashedHeight uint) (*getter.OPIOrdGetterTest, RuntimeArguments) {
	arguments := RuntimeArguments{
		EnableService:        false,
		EnableCommittee:      false,
		EnableStateRootCache: false,
		EnableTest:           false,
		TestBlockHeightLimit: 0,
	}

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
	gd := getter.DatabaseConfig(GlobalConfig.Database)
	g, err := getter.NewOPIOrdGetterTest(&gd, arguments.TestBlockHeightLimit, hashedHeight)

	if err != nil {
		log.Fatalf("Failed to catchup the latest state: %v", err)
	}

	return g, arguments
}
