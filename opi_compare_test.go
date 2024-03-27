package main

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/RiemaLabs/modular-indexer-committee/ord"
	"github.com/RiemaLabs/modular-indexer-committee/ord/stateless"
)

func TestOPI(t *testing.T) {
	var latestHeight uint = stateless.BRC20StartHeight + ord.BitcoinConfirmations
	records, err := stateless.LoadOPIRecords("./data/785000-ordi.csv")
	if err != nil {
		log.Fatalf(fmt.Sprintf("error happened: %v", err))
	}
	ordGetterTest, arguments := loadMain()
	queue, err := catchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, latestHeight)
	if err != nil {
		log.Fatalf(fmt.Sprintf("error happened: %v", err))
	}
	ordGetterTest.LatestBlockHeight = latestHeight
	go serviceStage(ordGetterTest, &arguments, queue, 500*time.Millisecond)
	for {
		if ordGetterTest.LatestBlockHeight == queue.LatestHeight() {
			queue.Header.VerifyState(&records)
			log.Printf("Block: %d is verfied!\n", ordGetterTest.LatestBlockHeight)
			ordGetterTest.LatestBlockHeight++
		}
		if ordGetterTest.LatestBlockHeight >= 781000 {
			os.Exit(0)
		}
		time.Sleep(500 * time.Millisecond)
	}
}
