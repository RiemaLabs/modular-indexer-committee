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
	var catchupHeight uint = 780000
	ordGetterTest, arguments := loadMain()
	queue, _ := CatchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, catchupHeight)
	ordGetterTest.LatestBlockHeight = catchupHeight

	startTime := time.Now()
	mockService(ordGetterTest, queue, 3) // partially update, some history still remain
	elapsed := time.Since(startTime)
	log.Printf("Using Time %s\n", elapsed)

	startTime = time.Now()
	mockService(ordGetterTest, queue, 10) // all update, no historical record stays
	elapsed = time.Since(startTime)
	log.Printf("Using Time %s\n", elapsed)
}

func mockService(getter getter.OrdGetter, queue *stateless.Queue, upHeight uint) {
	curHeight := queue.LatestHeight()
	latestHeight := curHeight + upHeight
	if curHeight < latestHeight {
		queue.Lock()
		err := queue.Update(getter, latestHeight)
		queue.Unlock()
		if err != nil {
			log.Fatalf("Failed To Update The Queue: %v", err)
		}
	}
	bytes := queue.Header.Root.Commit().Bytes()
	commitment := base64.StdEncoding.EncodeToString(bytes[:])
	log.Printf("Header's Commitment Is %s", commitment)
}
