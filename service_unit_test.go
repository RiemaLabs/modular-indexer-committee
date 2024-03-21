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
	getter, _ := loadMain()
	queue := loadCatchUp(getter, 780000, nil)

	startTime := time.Now()
	loadService(getter, queue, 3, nil) // partially update, some history still remain
	elapsed := time.Since(startTime)
	log.Printf("Using time %s\n", elapsed)

	startTime = time.Now()
	loadService(getter, queue, 10, nil) // all update, no historical record stays
	elapsed = time.Since(startTime)
	log.Printf("Using time %s\n", elapsed)
}

func loadService(getter getter.OrdGetter, queue *stateless.Queue, latestHeight uint, records *stateless.OPIRecords) {
	curHeight := queue.LatestHeight()

	if curHeight < latestHeight {
		queue.Lock()
		err := queue.DebugUpdateStrong(getter, latestHeight, records)
		// err := queue.DebugUpdate(getter, latestHeight) // For other cases
		queue.Unlock()
		if err != nil {
			log.Fatalf("Failed to update the queue: %v", err)
		}
	}

	// Hash and Height logging
	log.Printf("With header's height at %d, and header's hash to be %s", queue.Header.Height, queue.Header.Hash)

	// Commitment logging
	bytes := queue.Header.Root.Commit().Bytes()
	commitment := base64.StdEncoding.EncodeToString(bytes[:])
	log.Printf("Header's commitment is %s", commitment)
	queue.Println()
}
