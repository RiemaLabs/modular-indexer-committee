package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/RiemaLabs/indexer-committee/ord"
	"github.com/RiemaLabs/indexer-committee/ord/stateless"
)

func TestOPI(t *testing.T) {
	var latestHeight uint = stateless.BRC20StartHeight + ord.BitcoinConfirmations
	records, err := stateless.LoadOPIRecords("./data/785000-ordi.csv")
	if err != nil {
		panic(err)
	}
	ordGetterTest, arguments := loadMain()
	queue, err := catchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, latestHeight)
	if err != nil {
		panic(err)
	}
	ordGetterTest.LatestBlockHeight = latestHeight
	go serviceStage(ordGetterTest, &arguments, queue, 500*time.Millisecond)
	for {
		if ordGetterTest.LatestBlockHeight == queue.LatestHeight() {
			queue.Lock()
			queue.Header.DebugState(&records)
			fmt.Printf("Block: %d is verfied!\n", ordGetterTest.LatestBlockHeight)
			ordGetterTest.LatestBlockHeight++
			queue.Unlock()
		}
		if ordGetterTest.LatestBlockHeight == 785000 {
			os.Exit(0)
		}
		time.Sleep(500 * time.Millisecond)
	}
}
