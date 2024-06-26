package main

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/RiemaLabs/modular-indexer-committee/ord"
	"github.com/RiemaLabs/modular-indexer-committee/ord/stateless"
)

func Test_OPI(t *testing.T) {
	var latestHeight uint = stateless.BRC20StartHeight + ord.BitcoinConfirmations
	records, err := stateless.LoadOPIRecords("./data/785000-ordi.csv")
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
}
