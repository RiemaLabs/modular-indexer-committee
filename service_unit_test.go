package main

import (
	"encoding/base64"
	"log"
	"testing"
	"time"

	"github.com/RiemaLabs/indexer-committee/ord/getter"
	"github.com/RiemaLabs/indexer-committee/ord/stateless"
)

func TestService(t *testing.T) {
	var catchupHeight uint = 780000
	ordGetterTest, arguments := loadMain()
	queue, _ := catchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, catchupHeight)
	ordGetterTest.LatestBlockHeight = catchupHeight

	startTime := time.Now()
	loadService(ordGetterTest, queue, 3, nil) // partially update, some history still remain
	elapsed := time.Since(startTime)
	log.Printf("Using time %s\n", elapsed)

	startTime = time.Now()
	loadService(ordGetterTest, queue, 10, nil) // all update, no historical record stays
	elapsed = time.Since(startTime)
	log.Printf("Using time %s\n", elapsed)
}

func loadService(getter getter.OrdGetter, queue *stateless.Queue, upHeight uint, records *stateless.OPIRecords) {
	curHeight := queue.LastestHeight()
	latestHeight := curHeight + upHeight
	if curHeight < latestHeight {
		queue.Lock()
		err := queue.Update(getter, latestHeight, records)
		queue.Unlock()
		if err != nil {
			log.Fatalf("Failed to update the queue: %v", err)
		}
	}
	bytes := queue.Header.Root.Commit().Bytes()
	commitment := base64.StdEncoding.EncodeToString(bytes[:])
	log.Printf("With header's height at %d, and header's hash to be %s", queue.Header.Height, queue.Header.Hash)
	log.Printf("Header's commitment is %s", commitment)
}
