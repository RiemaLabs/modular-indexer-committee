package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"runtime/debug"
	"testing"
	"time"

	"github.com/RiemaLabs/indexer-committee/ord"
	"github.com/RiemaLabs/indexer-committee/ord/getter"
	"github.com/RiemaLabs/indexer-committee/ord/stateless"
)

func TestCatchUp(t *testing.T) {
	loadCatchUp()
}

func loadCatchUp() *stateless.Queue {
	getter, arguments := loadMain()

	var latestHeight uint = 780000
	initHeight := stateless.BRC20StartHeight - 1
	catchupHeight := latestHeight - ord.BitcoinConfirmations

	header := stateless.LoadHeader(arguments.EnableStateRootCache, initHeight)
	curHeight := header.Height

	if catchupHeight > curHeight {
		log.Printf("Fast catchup to the lateset block height! From %d to %d \n", curHeight, catchupHeight)
	}
	startTime := time.Now()
	for i := curHeight + 1; i <= catchupHeight; i++ {
		brc20Transfer, err := getter.GetOrdTransfers(i)
		if err != nil {
			log.Printf("Critical Error when fetch Transfers at heigth %d", i)
		}
		stateless.Exec(&header, brc20Transfer)
		if i%1000 == 0 {
			log.Printf("Blocks: %d / %d \n", i, catchupHeight)
			if arguments.EnableStateRootCache {
				err := stateless.StoreHeader(header, header.Height-2000)
				if err != nil {
					log.Printf("Failed to store the cache at height: %d", i)
				}
			}
		}
		header.Paging(getter, false, stateless.NodeResolveFn)
	}

	// Time logging
	elapsed := time.Since(startTime)
	elapsedSeconds := float64(elapsed) / float64(time.Second)
	averageTime := elapsedSeconds / float64(latestHeight-stateless.BRC20StartHeight)
	log.Printf("Successfully Updating from %d to %d", stateless.BRC20StartHeight, latestHeight)
	log.Printf("Using time %s, and %f perline on average", elapsed, averageTime)

	// Hash and Height logging
	log.Printf("With header's height at %d, and header's hash to be %s", header.Height, header.Hash)

	// Commitment logging
	bytes := header.Root.Commit().Bytes()
	commitment := base64.StdEncoding.EncodeToString(bytes[:])
	log.Printf("Header's commitment is %s", commitment)

	queue, err := stateless.NewQueues(getter, &header, true, catchupHeight)
	if err != nil {
		log.Printf("Critical Error when generating New queue")
	}

	// Queue inner logging, checking its header and History Component
	queue.Println()

	return queue
}

func loadMain() (*getter.OPIOrdGetter, RuntimeArguments) {
	arguments := RuntimeArguments{
		EnableService:        false,
		EnableCommittee:      false,
		EnableStateRootCache: true,
	}
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
		log.Fatalf("Failed to catchup the latest state: %v", err)
	}

	return getter, arguments

}
