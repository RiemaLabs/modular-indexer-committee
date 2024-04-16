package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RiemaLabs/modular-indexer-committee/apis"
	"github.com/RiemaLabs/modular-indexer-committee/ord/getter"
	"github.com/RiemaLabs/modular-indexer-committee/ord/stateless"
	"github.com/gin-gonic/gin"
)

func duplicateOrdTransfer(ordTransfers []getter.OrdTransfer) []getter.OrdTransfer {
	dup := make([]getter.OrdTransfer, len(ordTransfers))
	for i, transfer := range ordTransfers {
		dup[i] = transfer
		if i == len(ordTransfers)-1 {
			dup = append(dup, transfer)
		}
	}
	return dup
}

func omitOrdTransfer(ordTransfers []getter.OrdTransfer) []getter.OrdTransfer {
	if len(ordTransfers) == 0 {
		return ordTransfers
	}
	return ordTransfers[:len(ordTransfers)-1]
}

func changeLastTransactionAmount(ordTransfers []getter.OrdTransfer) []getter.OrdTransfer {
	if len(ordTransfers) == 0 {
		return ordTransfers
	}
	var js map[string]string
	json.Unmarshal(ordTransfers[len(ordTransfers)-1].Content, &js)
	amountString, ok := js["amt"]
	if !ok {
		return ordTransfers
	}
	amountString = amountString + "000"
	js["amt"] = amountString
	content, _ := json.Marshal(js)
	ordTransfers[len(ordTransfers)-1].Content = content
	return ordTransfers
}

func verifyMaliciousCheckpoint(maliciousFunc func(ordTransfers []getter.OrdTransfer) []getter.OrdTransfer, catchupHeight uint, t *testing.T) (time.Duration, time.Duration, bool) {
	ordGetter, arguments := loadMain()
	postQueue, _ := CatchupStage(ordGetter, &arguments, stateless.BRC20StartHeight-1, catchupHeight)

	ordTransfer := postQueue.Header.OrdTrans

	prevQueue, _ := CatchupStage(ordGetter, &arguments, stateless.BRC20StartHeight-1, catchupHeight-1)

	// Get malicious commit
	header := prevQueue.Header
	maliciousOrdTransfer := maliciousFunc(ordTransfer)
	headerMalicious := stateless.Header{
		Height:         header.Height,
		Hash:           header.Hash,
		Root:           header.Root,
		IntermediateKV: header.IntermediateKV,
		KV:             header.KV,
		Access:         header.Access,
	}
	stateless.Exec(&headerMalicious, maliciousOrdTransfer, catchupHeight)
	headerMalicious.Paging(ordGetter, false, stateless.NodeResolveFn)
	maliciousCommit := headerMalicious.Root.Commit()

	// Get correct proof
	// Set gin as test mode
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/v1/brc20_verifiable/latest_state_proof", func(c *gin.Context) {
		apis.GetLatestStateProof(c, postQueue)
	})
	// create test server
	ts := httptest.NewServer(r)
	defer ts.Close()
	req, err := http.NewRequest("GET", ts.URL+"/v1/brc20_verifiable/latest_state_proof", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	// check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("[TestGetLatestStateProof]", err)
	}
	// Get result
	var proof apis.Brc20VerifiableLatestStateProofResponse
	if err := json.Unmarshal(body, &proof); err != nil {
		log.Fatal("[TestGetLatestStateProof]", err)
	}

	lastIndex := len(postQueue.History) - 1
	preState, _ := stateless.Rollingback(postQueue.Header, &postQueue.History[lastIndex])
	startTime := time.Now()
	_, err = apis.GeneratePostRoot(preState.Commit(), catchupHeight, &proof)
	if err != nil {
		log.Fatal("With error: ", err)
	} else {
		log.Println("True checkpoint passed successfully")
	}
	endTime := time.Now()
	verifyCorrectTime := endTime.Sub(startTime)

	findError := false
	startTime = time.Now()
	_, err = apis.GeneratePostRoot(maliciousCommit, catchupHeight, &proof)
	if err != nil {
		findError = true
	}
	endTime = time.Now()
	verifyMaliciousTime := endTime.Sub(startTime)

	return verifyCorrectTime, verifyMaliciousTime, findError
}

// The following three tests are to verify the malicious checkpoint, return the time to verify the correct checkpoint and the malicious checkpoint, also return if the malicious checkpoint is found
func TestDuplicateCheckpoint(t *testing.T) {
	verifyCorrectTime, verifyMaliciousTime, findError := verifyMaliciousCheckpoint(duplicateOrdTransfer, 780000, t)
	log.Println("Verify Correct Commit time: ", verifyCorrectTime)
	log.Println("Verify Duplicated Commit time: ", verifyMaliciousTime)
	if findError {
		log.Println("Find malicious checkpoint successfully")
	} else {
		log.Println("Error not found")
	}
}

func TestOmitCheckpoint(t *testing.T) {
	verifyCorrectTime, verifyMaliciousTime, findError := verifyMaliciousCheckpoint(omitOrdTransfer, 780000, t)
	log.Println("Verify Correct Commit time: ", verifyCorrectTime)
	log.Println("Verify Omited Commit time: ", verifyMaliciousTime)
	if findError {
		log.Println("Find malicious checkpoint successfully")
	} else {
		log.Println("Error not found")
	}
}

func TestChangeCheckpoint(t *testing.T) {
	verifyCorrectTime, verifyMaliciousTime, findError := verifyMaliciousCheckpoint(changeLastTransactionAmount, 780000, t)
	log.Println("Verify Correct Commit time: ", verifyCorrectTime)
	log.Println("Verify Changed Commit time: ", verifyMaliciousTime)
	if findError {
		log.Println("Find malicious checkpoint successfully")
	} else {
		log.Println("Error not found")
	}
}
