package main

import (
	"encoding/base64"
	"log"
	"testing"
	"time"

	"github.com/RiemaLabs/indexer-committee/ord/stateless"
)

func TestCatchUp(t *testing.T) {
	var catchupHeight uint = 780000
	ordGetterTest, arguments := loadMain()
	startTime := time.Now()
	queue, _ := catchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, catchupHeight)
	if queue.Header.Height != catchupHeight {
		log.Println("Queue header not updated correctly")
	}
	ordGetterTest.LatestBlockHeight = catchupHeight
	elapsed := time.Since(startTime)
	elapsedSeconds := float64(elapsed) / float64(time.Second)
	averageTime := elapsedSeconds / float64(catchupHeight-stateless.BRC20StartHeight)
	log.Printf("Successfully Updating from %d to %d", stateless.BRC20StartHeight, catchupHeight)
	log.Printf("Using time %s, and %f perline on average during CatchUp Stage", elapsed, averageTime)
	log.Printf("With header's height at %d, and header's hash to be %s", queue.Header.Height, queue.Header.Hash)

	// Commitment logging
	bytes := queue.Header.Root.Commit().Bytes()
	commitment := base64.StdEncoding.EncodeToString(bytes[:])
	log.Printf("Header's commitment is %s", commitment)
}
