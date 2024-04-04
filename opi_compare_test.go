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
	ordGetterTest, arguments := loadMain()
	queue, err := CatchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, latestHeight)
	if err != nil {
		log.Fatalf(fmt.Sprintf("error happened: %v", err))
	}
	ordGetterTest.LatestBlockHeight = latestHeight
	go ServiceStage(ordGetterTest, &arguments, queue, 10*time.Millisecond)
	for {
		if ordGetterTest.LatestBlockHeight == queue.LatestHeight() {
			queue.Header.VerifyState(&records)
			// output this line every 100 blocks
			if ordGetterTest.LatestBlockHeight%100 == 0 {
				log.Printf("Block: %d is verfied!\n", ordGetterTest.LatestBlockHeight)
			}
			ordGetterTest.LatestBlockHeight++
		}
		if ordGetterTest.LatestBlockHeight >= 780000 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
}
