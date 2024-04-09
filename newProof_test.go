package main

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/RiemaLabs/modular-indexer-committee/ord"
	"github.com/RiemaLabs/modular-indexer-committee/ord/stateless"
)

func Test_NewProof(t *testing.T) {
	var latestHeight uint = stateless.BRC20StartHeight + ord.BitcoinConfirmations
	ordGetterTest, arguments := loadMain()
	queue, err := CatchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, latestHeight)
	if err != nil {
		log.Fatalf(fmt.Sprintf("error happened: %v", err))
	}
	ordGetterTest.LatestBlockHeight = latestHeight
	go ServiceStage(ordGetterTest, &arguments, queue, 10*time.Millisecond)
	for {
		if ordGetterTest.LatestBlockHeight == queue.LatestHeight() {
			if queue.VerifyProof() {
				log.Printf("Block: %d is verified!\n", ordGetterTest.LatestBlockHeight)
			} else {
				log.Printf("Block: %d cannot pass verification!\n", ordGetterTest.LatestBlockHeight)
			}
			ordGetterTest.LatestBlockHeight++
		}
		if ordGetterTest.LatestBlockHeight >= 780000 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
}
