package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/RiemaLabs/modular-indexer-committee/apis"
	"github.com/RiemaLabs/modular-indexer-committee/ord/getter"
	"github.com/RiemaLabs/modular-indexer-committee/ord/stateless"
	"github.com/gin-gonic/gin"
)

type BlockVerifyData struct {
	Height              uint
	VerifyCorrectTime   time.Duration
	VarifyDuplicateTime time.Duration
	VarifyOmitTime      time.Duration
	VarifyChangeTime    time.Duration
}

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

func getCorrectProofResp(catchupHeight uint, t *testing.T) apis.Brc20VerifiableLatestStateProofResponse {
	ordGetterTest, arguments := loadMain()
	queue, _ := CatchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, catchupHeight)

	// Set gin as test mode
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/v1/brc20_verifiable/latest_state_proof", func(c *gin.Context) {
		apis.GetLatestStateProof(c, queue)
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
	var res apis.Brc20VerifiableLatestStateProofResponse
	if err := json.Unmarshal(body, &res); err != nil {
		log.Fatal("[TestGetLatestStateProof]", err)
	}

	return res
}

func verifyMaliciousCheckpoint(ordGetter getter.OrdGetter, arguments *RuntimeArguments, t *testing.T) []BlockVerifyData {
	blockVerifyData := []BlockVerifyData{}
	initHeight := stateless.BRC20StartHeight - 1
	latestHeight := uint(780000)
	// Fetch the latest block height.
	header := stateless.LoadHeader(arguments.EnableStateRootCache, initHeight)
	curHeight := header.Height

	log.Printf("Fast catchup to the lateset block height! From %d to %d \n", curHeight, latestHeight)

	// Create a channel to listen for SIGINT (Ctrl+C) signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	// Start to catch-up

	if latestHeight > curHeight {
		for i := curHeight + 1; i <= latestHeight; i++ {
			select {
			case <-sigChan:
				log.Printf("Saving cache file. Please don't force exit.")
				stateless.StoreHeader(header, header.Height-2000)
				os.Exit(0)
			default:
				ordTransfer, err := ordGetter.GetOrdTransfers(i)
				if err != nil {
					panic(err)
				}

				// Get duplicate commit
				duplicatedOrdTransfer := duplicateOrdTransfer(ordTransfer)
				headerDuplicate := stateless.Header{
					Height:         header.Height,
					Hash:           header.Hash,
					Root:           header.Root,
					IntermediateKV: header.IntermediateKV,
					KV:             header.KV,
					Access:         header.Access,
				}
				headerDuplicate.Lock()
				stateless.Exec(&headerDuplicate, duplicatedOrdTransfer, i)
				headerDuplicate.Paging(ordGetter, false, stateless.NodeResolveFn)
				headerDuplicate.Unlock()
				duplicatedCommit := headerDuplicate.Root.Commit()

				// Get omitted commit
				omittedOrdTransfer := omitOrdTransfer(ordTransfer)
				headerOmitted := stateless.Header{
					Height:         header.Height,
					Hash:           header.Hash,
					Root:           header.Root,
					IntermediateKV: header.IntermediateKV,
					KV:             header.KV,
					Access:         header.Access,
				}
				headerOmitted.Lock()
				stateless.Exec(&headerOmitted, omittedOrdTransfer, i)
				headerOmitted.Paging(ordGetter, false, stateless.NodeResolveFn)
				headerOmitted.Unlock()
				omittedCommit := headerOmitted.Root.Commit()

				// Get changed commit
				changedLastTransactionAmount := changeLastTransactionAmount(ordTransfer)
				headerChanged := stateless.Header{
					Height:         header.Height,
					Hash:           header.Hash,
					Root:           header.Root,
					IntermediateKV: header.IntermediateKV,
					KV:             header.KV,
					Access:         header.Access,
				}
				headerChanged.Lock()
				stateless.Exec(&headerChanged, changedLastTransactionAmount, i)
				headerChanged.Paging(ordGetter, false, stateless.NodeResolveFn)
				headerChanged.Unlock()
				changedCommit := headerChanged.Root.Commit()

				// Get correct commit
				header.Lock()
				stateless.Exec(header, ordTransfer, i)
				header.Paging(ordGetter, false, stateless.NodeResolveFn)
				header.Unlock()
				correctCommit := header.Root.Commit()

				// Get correct proof
				proof := getCorrectProofResp(i, t)

				// Verify the correct commit
				startTime := time.Now()
				apis.GeneratePostRoot(correctCommit, i, &proof)
				endTime := time.Now()
				VerifyCorrectTime := endTime.Sub(startTime)

				// Verify the duplicated commit
				startTime = time.Now()
				apis.GeneratePostRoot(duplicatedCommit, i, &proof)
				endTime = time.Now()
				VarifyDuplicateTime := endTime.Sub(startTime)

				// Verify the omitted commit
				startTime = time.Now()
				apis.GeneratePostRoot(omittedCommit, i, &proof)
				endTime = time.Now()
				VarifyOmitTime := endTime.Sub(startTime)

				// Verify the changed commit
				startTime = time.Now()
				apis.GeneratePostRoot(changedCommit, i, &proof)
				endTime = time.Now()
				VarifyChangeTime := endTime.Sub(startTime)

				blockVerifyData = append(blockVerifyData, BlockVerifyData{
					Height:              i,
					VerifyCorrectTime:   VerifyCorrectTime,
					VarifyDuplicateTime: VarifyDuplicateTime,
					VarifyOmitTime:      VarifyOmitTime,
					VarifyChangeTime:    VarifyChangeTime,
				})

				if i%1000 == 0 {
					log.Printf("Blocks: %d / %d \n", i, latestHeight)
					if arguments.EnableStateRootCache {
						err := stateless.StoreHeader(header, header.Height-2000)
						if err != nil {
							panic(err)
						}
					}
				}
			}
		}
	}

	return blockVerifyData
}

func TestMaliciousCheckpoint(t *testing.T) {
	ordGetterTest, arguments := loadMain()
	blockVerifyData := verifyMaliciousCheckpoint(ordGetterTest, &arguments, t)
	jsonData, err := json.MarshalIndent(blockVerifyData, "", "    ")
	if err != nil {
		log.Println("[Save to JSON] Error: ", err)
		return
	}

	fileName := "verify-malicious-checkpoint-data.json"
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
