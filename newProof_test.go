package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/RiemaLabs/modular-indexer-committee/ord"
	"github.com/RiemaLabs/modular-indexer-committee/ord/stateless"
	"github.com/ethereum/go-verkle"
)

func Test_NewProof(t *testing.T) {
	var latestHeight uint = stateless.BRC20StartHeight + ord.BitcoinConfirmations
	ordGetterTest, arguments := loadMain(782000)
	queue, err := CatchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, latestHeight)
	if err != nil {
		log.Fatalf(fmt.Sprintf("error happened: %v", err))
	}
	ordGetterTest.LatestBlockHeight = latestHeight
	go ServiceStage(ordGetterTest, &arguments, queue, 10*time.Millisecond)
	for {
		if ordGetterTest.LatestBlockHeight == queue.LatestHeight() {
			if VerifyProof(queue) {
				log.Printf("Block: %d is verified!\n", ordGetterTest.LatestBlockHeight)
			} else {
				log.Fatalf("Block: %d cannot pass verification!\n", ordGetterTest.LatestBlockHeight)
			}
			ordGetterTest.LatestBlockHeight++
		}
		if ordGetterTest.LatestBlockHeight >= 780000 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func VerifyProof(queue *stateless.Queue) bool {
	if queue.LastStateProof == nil {
		log.Println("queue.LastStateProof == nil")
		return true
	}
	vProof, _, err := verkle.SerializeProof(queue.LastStateProof)
	if err != nil {
		log.Println("[VerifyProof]: verkle.SerializeProof(queue.LastStateProof) failed")
		return false
	}
	vProofBytes, err := vProof.MarshalJSON()
	if err != nil {
		return false
	}
	finalproof := base64.StdEncoding.EncodeToString(vProofBytes[:])
	rollingbackProof := RollingbackProof(queue)
	if rollingbackProof == "" {
		return true
	}
	return finalproof == rollingbackProof
}

func RollingbackProof(queue *stateless.Queue) string {
	// copy most code from apis.GetLatestStateProof
	// and then return the finalproof
	lastIndex := len(queue.History) - 1
	postState := queue.Header.Root
	preState, keys := stateless.Rollingback(queue.Header, &queue.History[lastIndex])

	if len(keys) == 0 {
		log.Println("[RollingbackProof]: len(keys) == 0")
		return ""
	}

	proofOfKeys, _, _, _, err := verkle.MakeVerkleMultiProof(preState, postState, keys, stateless.NodeResolveFn)
	if err != nil {
		log.Printf("Failed to generate proof due to %v", err)
		return ""
	}

	vProof, _, err := verkle.SerializeProof(proofOfKeys)
	if err != nil {
		log.Printf("Failed to serialize proof due to %v", err)
		return ""
	}

	vProofBytes, err := vProof.MarshalJSON()
	if err != nil {
		log.Printf("Failed to marshal the proof to JSON due to %v", err)
		return ""
	}

	finalproof := base64.StdEncoding.EncodeToString(vProofBytes[:])
	return finalproof
}
