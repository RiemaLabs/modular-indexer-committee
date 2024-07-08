package main

import (
	"encoding/base64"
	"log"
	"testing"
	"time"

	"github.com/RiemaLabs/modular-indexer-committee/ord/getter"
	"github.com/RiemaLabs/modular-indexer-committee/ord/stateless"
)

func Test_ServiceStage(t *testing.T) {
	log.Println("Test_ServiceStage")
	var catchupHeight uint = 780000
	ordGetterTest, arguments := loadMain(782000)
	queue, _ := CatchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, catchupHeight)
	ordGetterTest.SetLatestBlockHeight(catchupHeight)

	startTime := time.Now()
	mockService(ordGetterTest, queue, 3) // partially update, some history still remain
	elapsed := time.Since(startTime)
	log.Printf("Using Time %s\n", elapsed)

	startTime = time.Now()
	mockService(ordGetterTest, queue, 10) // all update, no historical record stays
	elapsed = time.Since(startTime)
	log.Printf("Using Time %s\n", elapsed)
	stateless.CleanPath(stateless.VerkleDataPath)
}

func mockService(getter getter.OrdGetter, queue *stateless.Queue, upHeight uint) {
	curHeight := queue.LatestHeight()
	latestHeight := curHeight + upHeight
	if curHeight < latestHeight {
		err := queue.Update(getter, latestHeight)
		if err != nil {
			log.Fatalf("Failed To Update The Queue: %v", err)
		}
	}
	bytes := queue.Header.Root.VerkleTree.Commit().Bytes()
	commitment := base64.StdEncoding.EncodeToString(bytes[:])
	log.Printf("Header's Commitment at Height %d Is %s", latestHeight, commitment)
}
