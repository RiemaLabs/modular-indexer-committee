package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"testing"
	"time"

	"github.com/RiemaLabs/modular-indexer-committee/checkpoint"
	"github.com/RiemaLabs/modular-indexer-committee/ord"
	"github.com/RiemaLabs/modular-indexer-committee/ord/stateless"
	"github.com/ethereum/go-verkle"
)

type CheckpointData struct {
	Height        uint
	stateDiffTime time.Duration
	fullStateTime time.Duration
}

func TestCheckpointExperiment(t *testing.T) {
	checkpointData := NewCheckpointExperiment(790000, 790100)
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
	ordGetterTest, arguments := loadMain(endHeight)
	for i := startHeight; i <= endHeight; i++ {
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

func Test_LoadOrdTransfers(t *testing.T) {
	ord.GenerateOrdTransfers(790100)
}

func Test_LoadBRC20BlockHashes(t *testing.T) {
	ord.GenerateBRC20BlockHashes(790100)
}

func NewCheckpointExperiment(startHeight uint, endHeight uint) []CheckpointData {
	checkpointData := make([]CheckpointData, 0)
	// Calculate the next checkpoint using the previous checkpoint and stateDiff.
	// For each block height, start with the previous checkpoint and apply stateDiff to compute the next checkpoint.
	// Record the time taken to compute each checkpoint in milliseconds.
	getter, arguments := loadMain(uint(790100))
	queue, _ := CatchupStage(getter, &arguments, stateless.BRC20StartHeight-1, startHeight-1)
	indexerID := checkpoint.IndexerIdentification{
		URL:          GlobalConfig.Service.URL,
		Name:         arguments.CommitteeIndexerName,
		Version:      Version,
		MetaProtocol: GlobalConfig.Service.MetaProtocol,
	}
	log.Println("finish catchup")
	for i := startHeight; i <= endHeight; i++ {
		ordTransfer, err := getter.GetOrdTransfers(i)
		if err != nil {
			panic(err)
		}

		startTime := time.Now()
		stateless.Exec(queue.Header, ordTransfer, i)
		hash, err := getter.GetBlockHash(i - 1)
		if err != nil {
			panic(err)
		}
		newDiffState := stateless.DiffState{
			Height:       i - 1,
			Hash:         hash,
			Access:       queue.Header.Access,
			VerkleCommit: queue.Header.Root.Commit().Bytes(),
		}
		copy(queue.History[:], queue.History[1:])
		queue.History[len(queue.History)-1] = newDiffState
		commitment := base64.StdEncoding.EncodeToString(newDiffState.VerkleCommit[:])
		checkpoint.NewCheckpoint(&indexerID, i-1, hash, commitment)
		stateDiffTime := time.Since(startTime)

		startTime = time.Now()
		root := verkle.New()
		for k, v := range queue.Header.KV {
			root.Insert(k[:], v[:], stateless.NodeResolveFn)
		}
		commit := root.Commit().Bytes()
		commitment = base64.StdEncoding.EncodeToString(commit[:])
		hash, err = getter.GetBlockHash(i)
		if err != nil {
			panic(err)
		}
		checkpoint.NewCheckpoint(&indexerID, i-1, hash, commitment)
		fullStateTime := time.Since(startTime)

		queue.Header.OrdTrans = ordTransfer
		// header.Height ++
		queue.Header.Paging(getter, true, stateless.NodeResolveFn)

		checkpointData = append(checkpointData, CheckpointData{
			Height:        i,
			stateDiffTime: stateDiffTime / 1000000,
			fullStateTime: fullStateTime / 1000000,
		})
		log.Println("finish height: ", i)
	}
	return checkpointData
}
