package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/RiemaLabs/indexer-committee/ord"
	"github.com/RiemaLabs/indexer-committee/ord/getter"
	"github.com/RiemaLabs/indexer-committee/ord/stateless"
)

func TestCatchUp(t *testing.T) {
	ordGetter, _ := loadMain()
	loadCatchUp(ordGetter, 780000, nil)
}

func loadCatchUp(ordGetter getter.OrdGetter, latesetHeight uint, records *stateless.OPIRecords) *stateless.Queue {
	initHeight := stateless.BRC20StartHeight - 1
	catchupHeight := latesetHeight - ord.BitcoinConfirmations

	header := stateless.LoadHeader(false, initHeight)
	curHeight := header.Height

	if catchupHeight > curHeight {
		log.Printf("Fast catchup to the lateset block height! From %d to %d \n", curHeight, catchupHeight)
	}
	startTime := time.Now()
	for i := curHeight + 1; i <= catchupHeight-1; i++ {
		ordTransfers, err := ordGetter.GetOrdTransfers(i)
		if err != nil {
			panic(fmt.Errorf("critical Error when fetch Transfers at heigth %d", i))
		}
		stateless.Exec(&header, ordTransfers, i)
		// Height ++
		header.Paging(ordGetter, false, stateless.NodeResolveFn)
		if i%1000 == 0 {
			log.Printf("Blocks: %d / %d \n", i, catchupHeight)
		}
		if records != nil {
			header.DebugState(records)
		}
	}

	// Time logging
	elapsed := time.Since(startTime)
	elapsedSeconds := float64(elapsed) / float64(time.Second)
	averageTime := elapsedSeconds / float64(catchupHeight-stateless.BRC20StartHeight)
	log.Printf("Successfully Updating from %d to %d", stateless.BRC20StartHeight, catchupHeight)
	log.Printf("Using time %s, and %f perline on average", elapsed, averageTime)

	// Hash and Height logging
	log.Printf("With header's height at %d, and header's hash to be %s", header.Height, header.Hash)

	// Commitment logging
	bytes := header.Root.Commit().Bytes()
	commitment := base64.StdEncoding.EncodeToString(bytes[:])
	log.Printf("Header's commitment is %s", commitment)

	queue, err := stateless.NewQueues(ordGetter, &header, true, catchupHeight)
	if err != nil {
		log.Printf("Critical Error when generating New queue")
	}

	// Queue inner logging, checking its header and History Component
	queue.Println()

	return queue
}
