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

func TestAPI_GetLatestStateProof(t *testing.T) {
	loadGetLatestStateProof(uint(779980), t)
}

func TestAPI_GetLatestStateProof_ZeroTransfers(t *testing.T) {
	// There is no transaction at block 779940.
	loadGetLatestStateProof(uint(779940), t)
}

func loadGetLatestStateProof(catchupHeight uint, t *testing.T) {
	ordGetterTest, arguments := loadMain(782000)
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

	lastIndex := len(queue.History) - 1
	preState, _ := stateless.Rollingback(queue.Header, &queue.History[lastIndex])

	_, err = apis.GeneratePostRoot(preState.Commit(), queue.LatestHeight(), &res)
	if err != nil {
		log.Fatal("With error: ", err)
	}
}

func TestAPI_VerifyCurrentBalanceOfPkscript(t *testing.T) {
	loadVerifyCurrentBalanceOfPkscript("ordi", "5120409943cab2dee3c71940969a612c6ee65c57cad1f064ca8db4508dab49260ca3", uint(779960), t)
}

func loadVerifyCurrentBalanceOfPkscript(tick string, pkScript string, catchupHeight uint, t *testing.T) {
	ordGetterTest, arguments := loadMain(782000)
	queue, _ := CatchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, catchupHeight)

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

	req, err := http.NewRequest("GET", ts.URL+"/v1/brc20_verifiable/current_balance_of_pkscript?tick="+tick+"&pkscript="+pkScript, nil)
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

	log.Println("[res]: ", res.Result.AvailableBalance)

	_, err = apis.VerifyCurrentBalanceOfPkscript(queue.Header.Root.Commit(), tick, pkScript, &res)
	if err != nil {
		log.Fatalf("[TestVerifyCurrentBalanceOfPkscript] verify not right. At tick %s, pkScript %s, height %d", tick, pkScript, catchupHeight)
		log.Fatal("With error: ", err)
	}
}

func TestAPI_VerifyCurrentBalanceOfWallet(t *testing.T) {
	loadVerifyCurrentBalanceOfWallet("meme", "bc1prvqdfjku8359hk9uc2tdgg0xlwvsel2fjr9ysydmaas9x3kyzuvskuwmlq", uint(779980), t, 782000)
}

func loadVerifyCurrentBalanceOfWallet(tick string, wallet string, catchupHeight uint, t *testing.T, loadHeight uint) {
	ordGetterTest, arguments := loadMain(loadHeight)
	queue, _ := CatchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, catchupHeight)

	// Get current balance from api
	// Set gin as test mode
	gin.SetMode(gin.TestMode)

	// register route
	r := gin.Default()
	r.GET("/v1/brc20_verifiable/current_balance_of_wallet", func(c *gin.Context) {
		apis.GetCurrentBalanceOfWallet(c, queue)
	})

	// create test server
	ts := httptest.NewServer(r)
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL+"/v1/brc20_verifiable/current_balance_of_wallet?tick="+tick+"&wallet="+wallet, nil)
	if err != nil {
		t.Fatal("[TestVerifyCurrentBalanceOfWallet]", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal("[TestVerifyCurrentBalanceOfWallet]", err)
	}
	defer resp.Body.Close()

	// check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("[TestVerifyCurrentBalanceOfWallet]", err)
	}

	// Get result
	var res apis.Brc20VerifiableCurrentBalanceOfWalletResponse
	if err := json.Unmarshal(body, &res); err != nil {
		log.Fatal("[TestVerifyCurrentBalanceOfWallet]", err)
	}

	log.Println("[OverallBalance res]: ", res.Result.OverallBalance)
	log.Println("[AvailableBalance res]: ", res.Result.AvailableBalance)

	_, err = apis.VerifyCurrentBalanceOfWallet(queue.Header.Root.Commit(), tick, wallet, &res)
	if err != nil {
		// log.Fatalf("[TestVerifyCurrentBalanceOfWallet] verify not right. At tick %s, wallet %s, height %d", tick, wallet, catchupHeight)
		log.Fatal("With error: ", err)
	}
}
