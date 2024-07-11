package main

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/RiemaLabs/modular-indexer-committee/ord"
	"github.com/RiemaLabs/modular-indexer-committee/ord/stateless"
)

func Test_OKX(t *testing.T) {
	t.SkipNow() // This test can only be run in a single mode
	log.Println("Test_OKX")
	var latestHeight uint = stateless.BRC20StartHeight + ord.BitcoinConfirmations
	records, err := stateless.LoadORDRecords("./data/785000-ordi.csv")
	if err != nil {
		log.Fatalf(fmt.Sprintf("error happened: %v", err))
	}
	ordGetterTest, arguments := loadMain(782000)
	queue, err := CatchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, latestHeight)
	if err != nil {
		log.Fatalf(fmt.Sprintf("error happened: %v", err))
	}
	ordGetterTest.SetLatestBlockHeight(latestHeight)
	go ServiceStage(ordGetterTest, &arguments, queue, 10*time.Millisecond)
	for {
		curHeight, _ := ordGetterTest.GetLatestBlockHeight()
		if curHeight == queue.LatestHeight() {
			queue.Header.VerifyState(&records)
			log.Printf("Block: %d is verified!\n", curHeight)
			ordGetterTest.SetLatestBlockHeight(curHeight + 1)
		}
		newHeight, _ := ordGetterTest.GetLatestBlockHeight()
		if newHeight >= 780000 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	stateless.CleanPath(stateless.VerkleDataPath)
	log.Println("Test_OKX finished")
}
