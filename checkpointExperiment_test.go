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
	StateDiffTime time.Duration
	FullStateTime time.Duration
}

func Test_LoadOrdTransfers(t *testing.T) {
	ord.GenerateOrdTransfers(790100)
}

func Test_LoadBRC20BlockHashes(t *testing.T) {
	ord.GenerateBRC20BlockHashes(790100)
}

// The result of StateDiffTime and FullStateTime are in miliseconds.
func TestCheckpointExperiment(t *testing.T) {
	checkpointData := CheckpointExperiment(790000, 790100)
	log.Println("checkpointData: ", checkpointData)
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

		queue.Header.OrdTrans = ordTransfer
		// header.Height ++
		queue.Header.Paging(getter, true, stateless.NodeResolveFn)

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

		checkpointData = append(checkpointData, CheckpointData{
			Height:        i,
			StateDiffTime: stateDiffTime / 1000000,
			FullStateTime: fullStateTime / 1000000,
		})
		log.Println("finish height: ", i)
	}
	return checkpointData
}
