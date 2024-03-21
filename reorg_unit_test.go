package main

import (
	"encoding/base64"
	"log"

	"testing"
	"time"

	"github.com/RiemaLabs/indexer-committee/ord/getter"
	"github.com/RiemaLabs/indexer-committee/ord/stateless"
)

func TestReorg(t *testing.T) {
	var catchupHeight uint = 780000
	ordGetterTest, arguments := loadMain()
	queue, _ := catchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, catchupHeight)

	loadService(ordGetterTest, queue, 10, nil)

	loadReorg(ordGetterTest, queue, 0)
	loadReorg(ordGetterTest, queue, 1)
	loadReorg(ordGetterTest, queue, 2)
	loadReorg(ordGetterTest, queue, 3)
	loadReorg(ordGetterTest, queue, 4)
}

func loadReorg(getter getter.OrdGetter, queue *stateless.Queue, recovery uint) {
	log.Printf("Recover The Queue by %d Blocks!", recovery+1)
	startTime := time.Now()

	oldBytes := queue.Header.Root.Commit().Bytes()
	oldCommitment := base64.StdEncoding.EncodeToString(oldBytes[:])

	curHeight := queue.Header.Height
	recoveryTillHeight := curHeight - recovery
	queue.Recovery(getter, recoveryTillHeight)

	newBytes := queue.Header.Root.Commit().Bytes()
	newCommitment := base64.StdEncoding.EncodeToString(newBytes[:])

	if oldCommitment == newCommitment {
		log.Printf("Recover The Queue by %d Blocks Succeed!", recovery+1)
	} else {
		log.Printf("Recover The Queue by %d Blocks Failed Somewhere!", recovery+1)
	}
	elapsed := time.Since(startTime)
	log.Printf("Recovery One Block Using Time %s\n", elapsed)
}
