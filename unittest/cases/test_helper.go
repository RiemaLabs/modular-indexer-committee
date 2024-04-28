package cases

import (
	"encoding/json"
	"github.com/RiemaLabs/modular-indexer-committee"
	"log"
	"os"
	"runtime/debug"

	"github.com/RiemaLabs/modular-indexer-committee/ord/getter"
)

func loadMain(hashedHeight uint) (*getter.OPIOrdGetterTest, main.RuntimeArguments) {
	arguments := main.RuntimeArguments{
		EnableService:        false,
		EnableCommittee:      false,
		EnableStateRootCache: false,
		EnableTest:           false,
		TestBlockHeightLimit: 0,
	}

	// Get the version as a stamp for the checkpoint.
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		log.Fatalf("Failed to obtain build information.")
	}
	main.Version = bi.Main.Version

	// Get the configuration.
	configFile, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	err = json.Unmarshal(configFile, &main.GlobalConfig)
	if err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	// Use OPI database as the getter.
	gd := getter.DatabaseConfig(main.GlobalConfig.Database)
	getter, err := getter.NewOPIOrdGetterTest(&gd, arguments.TestBlockHeightLimit, hashedHeight)

	if err != nil {
		log.Fatalf("Failed to catchup the latest state: %v", err)
	}

	return getter, arguments
}
