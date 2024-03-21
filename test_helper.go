package main

import (
	"encoding/json"
	"log"
	"os"
	"runtime/debug"

	"github.com/RiemaLabs/indexer-committee/ord/getter"
)

func loadMain() (*getter.OPIOrdGetterTest, RuntimeArguments) {
	arguments := RuntimeArguments{
		EnableService:        false,
		EnableCommittee:      false,
		EnableStateRootCache: false,
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
	gd := getter.DatabaseConfig(GlobalConfig.Database)
	getter, err := getter.NewOPIOrdGetterTest(&gd)

	if err != nil {
		log.Fatalf("Failed to catchup the latest state: %v", err)
	}

	return getter, arguments
}
