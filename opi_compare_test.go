package main

import (
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
	go serviceStage(ordGetterTest, &arguments, queue, 1*time.Second)
	for {
		if ordGetterTest.LatestBlockHeight == queue.LastestHeight() {
			queue.Header.DebugState(&records)
			ordGetterTest.LatestBlockHeight++
		}
		if ordGetterTest.LatestBlockHeight == 785000 {
			os.Exit(0)
		}
		time.Sleep(1 * time.Second)
	}
}
