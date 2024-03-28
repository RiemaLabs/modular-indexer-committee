package main

import (
	"io"
	"log"
	"testing"

	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/RiemaLabs/modular-indexer-committee/apis"
	"github.com/RiemaLabs/modular-indexer-committee/ord/stateless"
	"github.com/gin-gonic/gin"
)

func TestGetLatestStateProof(t *testing.T) {
	loadGetLatestStateProof(uint(779000), t)
	// loadGetLatestStateProof(uint(780000), t)
}

func loadGetLatestStateProof(catchupHeight uint, t *testing.T) {
	ordGetterTest, arguments := loadMain()
	queue, _ := catchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, catchupHeight)
	// go apis.StartService(queue, arguments.EnableCommittee, arguments.EnableTest)

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
}

func TestVerifyCurrentBalanceOfPkscript(t *testing.T) {
	loadVerifyCurrentBalanceOfPkscript("ordi", "5120409943cab2dee3c71940969a612c6ee65c57cad1f064ca8db4508dab49260ca3", uint(779960), t)
}

func loadVerifyCurrentBalanceOfPkscript(tick string, pkScript string, catchupHeight uint, t *testing.T) {
	ordGetterTest, arguments := loadMain()
	queue, _ := catchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, catchupHeight)

	// Get current balance from api
	// Set gin as test mode
	gin.SetMode(gin.TestMode)

	// register route
	r := gin.Default()
	r.GET("/v1/brc20_verifiable/current_balance_of_pkscript", func(c *gin.Context) {
		apis.GetCurrentBalanceOfPkscript(c, queue)
	})

	// create test server
	ts := httptest.NewServer(r)
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL+"/v1/brc20_verifiable/current_balance_of_pkscript?tick="+tick+"pkScript="+pkScript, nil)
	if err != nil {
		t.Fatal("[TestVerifyCurrentBalanceOfPkscript]", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal("[TestVerifyCurrentBalanceOfPkscript]", err)
	}
	defer resp.Body.Close()

	// check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("[TestVerifyCurrentBalanceOfPkscript]", err)
	}

	// Get result
	var res apis.Brc20VerifiableCurrentBalanceOfPkscriptResponse
	if err := json.Unmarshal(body, &res); err != nil {
		log.Fatal("[TestVerifyCurrentBalanceOfPkscript]", err)
	}

	log.Println("[res]: ", res)

	lastHistory := queue.History[len(queue.History)-1]
	preState, _, _ := stateless.Rollingback(queue.Header, &lastHistory)
	_, err = apis.VerifyCurrentBalanceOfPkscript(preState.Commit(), tick, pkScript, &res)
	if err != nil {
		log.Fatalf("[TestVerifyCurrentBalanceOfPkscript] verify not right. At tick %s, pkScript %s, height %d", tick, pkScript, catchupHeight)
		log.Fatal("With error: ", err)
	}
}
