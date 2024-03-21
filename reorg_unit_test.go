package main

import (
	"encoding/base64"
	"log"

	"testing"
	"time"

	"github.com/RiemaLabs/indexer-committee/ord"
	"github.com/RiemaLabs/indexer-committee/ord/getter"
	"github.com/RiemaLabs/indexer-committee/ord/stateless"
)

func TestReorg(t *testing.T) {
	var catchupHeight uint = stateless.BRC20StartHeight + ord.BitcoinConfirmations
	ordGetterTest, arguments := loadMain()
	queue, _ := catchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, catchupHeight)

	loadReorg(ordGetterTest, queue, 1)

	mockService(ordGetterTest, queue, 10)

	loadReorg(ordGetterTest, queue, 2)
	loadReorg(ordGetterTest, queue, 3)
	loadReorg(ordGetterTest, queue, 4)
	loadReorg(ordGetterTest, queue, 5)
	loadReorg(ordGetterTest, queue, 6)
}

func loadReorg(getter getter.OrdGetter, queue *stateless.Queue, recovery uint) {
	startTime := time.Now()

	oldCommitments := make([]string, 0)

	for _, h := range queue.History {
		oldBytes := h.VerkleCommit
		oldCommitment := base64.StdEncoding.EncodeToString(oldBytes[:])
		oldCommitments = append(oldCommitments, oldCommitment)
	}

	curHeight := queue.Header.Height
	// reorgHeights means that the blockHash of this height changed.
	reorgHeight := curHeight - recovery + 1
	queue.Recovery(getter, reorgHeight)

	for i, h := range queue.History {
		newBytes := h.VerkleCommit
		newCommitment := base64.StdEncoding.EncodeToString(newBytes[:])
		oldCommitment := oldCommitments[i]
		if oldCommitment != newCommitment {
			log.Fatalf("Reorganize the queue by %d blocks failed!", recovery)
		}
	}
	elapsed := time.Since(startTime)
	log.Printf("Reorganize the queue by %d blocks succeed!", recovery)
	log.Printf("Timecost: %s\n", elapsed)
}
