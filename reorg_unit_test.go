package main

import (
	"log"
	"encoding/base64"
	// "time"
	"testing"

	"github.com/RiemaLabs/indexer-committee/ord/getter"
	"github.com/RiemaLabs/indexer-committee/ord/stateless"
)

func TestReorg(t *testing.T) {
	getter, _ := loadMain()
	queue := loadCatchUp()

	loadService(getter, queue, 3)
	// Try to recover Root by 1/2/6, and then recover the queue, and remember to compare the commitment
	loadReorg(getter, queue, 0)
	loadReorg(getter, queue, 1)
	loadReorg(getter, queue, 2)
	loadReorg(getter, queue, 3)
	loadReorg(getter, queue, 4) // at most
	// loadReorg(getter, queue, 5)
}

func loadReorg(getter getter.OrdGetter, queue *stateless.Queue, recovery uint) {
	log.Printf("Recover the queue by %d blocks!", recovery+1)
	// Get Old commitment and print old queue info
	oldBytes := queue.Header.Root.Commit().Bytes()
	oldCommitment := base64.StdEncoding.EncodeToString(oldBytes[:])
	queue.Println()

	// turn back the queue and recover back
	curHeight := queue.Header.Height
	recoveryTillHeight := curHeight - recovery
	queue.Recovery(getter, recoveryTillHeight)

	// Get New commitment and print new queue info and compare with the old
	newBytes := queue.Header.Root.Commit().Bytes()
	newCommitment := base64.StdEncoding.EncodeToString(newBytes[:])
	// queue.Println()

	if oldCommitment == newCommitment {
		log.Print("Great! Recovery succeed!")
	} else {
		log.Print("Recovery failed somewhere!")
	}
}