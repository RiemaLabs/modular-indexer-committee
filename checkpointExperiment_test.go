package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"testing"
	"time"

	"github.com/RiemaLabs/modular-indexer-committee/checkpoint"
	"github.com/RiemaLabs/modular-indexer-committee/ord/stateless"
	"github.com/ethereum/go-verkle"
)

type CheckpointData struct {
	Height        uint
	stateDiffTime time.Duration
	fullStateTime time.Duration
}

func TestCheckpointExperiment(t *testing.T) {
	checkpointData := CheckpointExperiment(790000, 790100)
	jsonData, err := json.MarshalIndent(checkpointData, "", "    ")
	if err != nil {
		log.Println("[Save to JSON] Error: ", err)
		return
	}

	fileName := "checkpoint-experiment.json"
	file, err := os.Create(fileName)
	if err != nil {
		log.Println("[Create File] Error", err)
		return
	}
	defer file.Close()
	_, err = file.Write(jsonData)
	if err != nil {
		log.Println("[Write File] Error", err)
		return
	}
}

func CheckpointExperiment(startHeight uint, endHeight uint) []CheckpointData {
	checkpointData := make([]CheckpointData, 0)
	// Calculate the next checkpoint using the previous checkpoint and stateDiff.
	// For each block height, start with the previous checkpoint and apply stateDiff to compute the next checkpoint.
	// Record the time taken to compute each checkpoint in milliseconds.
	for i := startHeight; i <= endHeight; i++ {
		ordGetterTest, arguments := loadMain(782000)
		queue, _ := CatchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, i)
		indexerID := checkpoint.IndexerIdentification{
			URL:          GlobalConfig.Service.URL,
			Name:         arguments.CommitteeIndexerName,
			Version:      Version,
			MetaProtocol: GlobalConfig.Service.MetaProtocol,
		}
		// Calculate the next checkpoint using the previous checkpoint and stateDiff.
		startTime := time.Now()
		log.Println(len(queue.History), queue.History)
		stateDiff := queue.History[len(queue.History)-1]
		commitment := base64.StdEncoding.EncodeToString(stateDiff.VerkleCommit[:])
		checkpoint.NewCheckpoint(&indexerID, stateDiff.Height, stateDiff.Hash, commitment)
		stateDiffTime := time.Since(startTime)

		// Calculate the next checkpoint using the full state.
		startTime = time.Now()
		root := verkle.New()
		for k, v := range queue.Header.KV {
			root.Insert(k[:], v[:], stateless.NodeResolveFn)
		}
		commit := root.Commit().Bytes()
		commitment = base64.StdEncoding.EncodeToString(commit[:])
		hash, err := ordGetterTest.GetBlockHash(i)
		if err != nil {
			panic(err)
		}
		checkpoint.NewCheckpoint(&indexerID, i, hash, commitment)
		fullStateTime := time.Since(startTime)

		checkpointData = append(checkpointData, CheckpointData{
			Height:        i,
			stateDiffTime: stateDiffTime / 1000000,
			fullStateTime: fullStateTime / 1000000,
		})
	}
	return checkpointData
}

func TestCatchup(t *testing.T) {
	ordGetterTest, arguments := loadMain(782000)
	queue, _ := CatchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, uint(790000))
	log.Println(len(queue.History), queue.History)
}
