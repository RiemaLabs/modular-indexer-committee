package main

import (
	"path/filepath"
	"strconv"
	"testing"

	"github.com/RiemaLabs/modular-indexer-committee/ord"
	"github.com/RiemaLabs/modular-indexer-committee/ord/stateless"
)

func Test_Serialization(t *testing.T) {
	var catchupHeight uint = 780050
	ordGetterTest, arguments := loadMain(782000)
	queue, _ := CatchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, catchupHeight)
	err := stateless.StoreHeader(queue.Header, catchupHeight)
	if err != nil {
		t.Errorf("Error storing header: %v", err)
	}
}

func Test_CleanPath(t *testing.T) {
	var catchupHeight uint = 780050
	cleanPath := filepath.Join(stateless.CachePath, strconv.Itoa(int(catchupHeight))+".dat")
	err := stateless.CleanPath(cleanPath)
	if err != nil {
		t.Errorf("Error cleaning path: %v", err)
	}
}

func Test_Deserialization(t *testing.T) {
	var catchupHeight uint = 780050 + ord.BitcoinConfirmations
	ordGetterTest, arguments := loadMain(782000)
	arguments.EnableStateRootCache = true
	// should load the generated DB
	queue2, _ := CatchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, catchupHeight)
	if queue2.Header.Height != catchupHeight {
		t.Errorf("Header height not equal")
	}
}
func Test_Recover(t *testing.T) {
	var catchupHeight uint = 780050 + ord.BitcoinConfirmations
	ordGetterTest, arguments := loadMain(782000)
	arguments.EnableStateRootCache = true
	queue, _ := CatchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, catchupHeight)
	// stateless.StoreHeader(queue.Header, catchupHeight)
	// should load the generated DB
	queue.Recovery(ordGetterTest, catchupHeight) // this will abandon current queue
	mockService(ordGetterTest, queue, 10)        // test if queue can still grow
}

// func loadReorg(getter getter.OrdGetter, queue *stateless.Queue, recovery uint) {
// 	startTime := time.Now()

// 	oldCommitments := make([]string, 0)

// 	for _, h := range queue.History {
// 		oldBytes := h.VerkleCommit
// 		oldCommitment := base64.StdEncoding.EncodeToString(oldBytes[:])
// 		oldCommitments = append(oldCommitments, oldCommitment)
// 	}

// 	curHeight := queue.Header.Height
// 	// reorgHeights means that the blockHash of this height changed.
// 	reorgHeight := curHeight - recovery + 1
// 	_ = queue.Recovery(getter, reorgHeight)

// 	for i, h := range queue.History {
// 		newBytes := h.VerkleCommit
// 		newCommitment := base64.StdEncoding.EncodeToString(newBytes[:])
// 		oldCommitment := oldCommitments[i]
// 		if oldCommitment != newCommitment {
// 			log.Fatalf("Reorganize the queue by %d blocks failed!", recovery)
// 		}
// 		log.Printf("Commitment at height %d: %s", h.Height, newCommitment)
// 	}
// 	b := queue.Header.Root.VerkleTree.Commit().Bytes()
// 	latestCommitment := base64.StdEncoding.EncodeToString(b[:])
// 	log.Printf("Commitment at height %d: %s", queue.Header.Height, latestCommitment)
// 	elapsed := time.Since(startTime)
// 	log.Printf("Reorganize the queue by %d blocks succeed!", recovery)
// 	log.Printf("Timecost: %s\n", elapsed)
// }
