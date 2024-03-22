package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"strconv"
	"testing"
	"time"

	"github.com/RiemaLabs/indexer-committee/checkpoint"
	"github.com/RiemaLabs/indexer-committee/ord/getter"
	"github.com/RiemaLabs/indexer-committee/ord/stateless"
)

func TestService(t *testing.T) {
	var catchupHeight uint = 780000
	ordGetterTest, arguments := loadMain()
	queue, _ := catchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, catchupHeight)
	ordGetterTest.LatestBlockHeight = catchupHeight

	startTime := time.Now()
	mockService(ordGetterTest, queue, 3) // partially update, some history still remain
	elapsed := time.Since(startTime)
	log.Printf("Using Time %s\n", elapsed)

	startTime = time.Now()
	mockService(ordGetterTest, queue, 10) // all update, no historical record stays
	elapsed = time.Since(startTime)
	log.Printf("Using time %s\n", elapsed)

	log.Printf("?\n")

	var history = make(map[string]checkpoint.UploadRecord)

	latestHistory := stateless.DiffState{
		Height:       queue.Header.Height,
		Hash:         queue.Header.Hash,
		VerkleCommit: queue.Header.Root.Commit().Bytes(),
		Diff:         stateless.DiffList{},
	}
	hs := make([]*stateless.DiffState, 0)
	for _, i := range queue.History {
		hs = append(hs, &i)
	}
	hs = append(hs, &latestHistory)
	for _, i := range hs {
		key := fmt.Sprintf("%d", i.Height) + i.Hash
		if curRecord, found := history[key]; !(found && curRecord.Success) {
			indexerID := checkpoint.IndexerIdentification{
				URL:          GlobalConfig.Service.URL,
				Name:         GlobalConfig.Service.Name,
				Version:      Version,
				MetaProtocol: GlobalConfig.Service.MetaProtocol,
			}
			commitment := base64.StdEncoding.EncodeToString(i.VerkleCommit[:])
			c := checkpoint.NewCheckpoint(&indexerID, i.Height, i.Hash, commitment)
			err := error(nil)
			timeout := time.Duration(GlobalConfig.Report.Timeout) * time.Millisecond
			if GlobalConfig.Report.Method == "s3" {
				err = checkpoint.UploadCheckpointByS3(&indexerID, &c, GlobalConfig.Report.S3.Region, GlobalConfig.Report.S3.Bucket, timeout)
			} else if GlobalConfig.Report.Method == "da" {
				err = checkpoint.UploadCheckpointByDA(&indexerID, &c,
					GlobalConfig.Report.Da.RPC, GlobalConfig.Report.Da.PrivateKey, GlobalConfig.Report.Da.InviteCode, GlobalConfig.Report.Da.NamespaceID,
					arguments.NetWork, timeout,
				)
			}
			if err != nil {
				log.Fatalf("Unable to upload the checkpoint because: %v", err)
			}
			history[key] = checkpoint.UploadRecord{
				Success: true,
			}

			objectKey := fmt.Sprintf("test/checkpoint-%s-%s-%s-%s.json",
				indexerID.Name, indexerID.MetaProtocol, strconv.FormatUint(uint64(queue.Header.Height), 10), queue.Header.Hash)
			err = checkpoint.DownloadCheckpointByS3(indexerID, objectKey, GlobalConfig.Report.S3.Region, GlobalConfig.Report.S3.Bucket, timeout)
		}
	}
}

func mockService(getter getter.OrdGetter, queue *stateless.Queue, upHeight uint) {
	curHeight := queue.LatestHeight()
	latestHeight := curHeight + upHeight
	if curHeight < latestHeight {
		queue.Lock()
		err := queue.Update(getter, latestHeight)
		queue.Unlock()
		if err != nil {
			log.Fatalf("Failed To Update The Queue: %v", err)
		}
	}
	bytes := queue.Header.Root.Commit().Bytes()
	commitment := base64.StdEncoding.EncodeToString(bytes[:])
	log.Printf("Header's Commitment Is %s", commitment)
}
