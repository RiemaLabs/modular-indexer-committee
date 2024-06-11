package apis

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ethereum/go-verkle"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"

	"github.com/RiemaLabs/modular-indexer-committee/internal/metrics"
	"github.com/RiemaLabs/modular-indexer-committee/ord"
	"github.com/RiemaLabs/modular-indexer-committee/ord/stateless"
)

func GetAllBalances(queue *stateless.Queue, tick string, wallet ord.Wallet) ([]byte, []byte, Brc20VerifiableCurrentBalanceOfWalletResult) {
	availKey, overKey, availableBalance, overallBalance := stateless.GetBalances(queue.Header, tick, wallet)
	availableBalanceStr := availableBalance.String()
	overallBalanceStr := overallBalance.String()

	result := Brc20VerifiableCurrentBalanceOfWalletResult{
		AvailableBalance: availableBalanceStr,
		OverallBalance:   overallBalanceStr,
		Wallet:           string(wallet),
	}

	return availKey, overKey, result
}

func GetCurrentBalanceOfWallet(c *gin.Context, queue *stateless.Queue) {
	tick := c.DefaultQuery("tick", "")
	wallet := c.DefaultQuery("wallet", "")

	availKey, overKey, result := GetAllBalances(queue, tick, ord.Wallet(wallet))

	keys := [][]byte{availKey, overKey}

	proof, _, _, _, err := verkle.MakeVerkleMultiProof(queue.Header.Root, nil, keys, stateless.NodeResolveFn)
	if err != nil {
		errStr := fmt.Sprintf("Failed to generate proof due to %v", err)
		c.JSON(http.StatusInternalServerError, Brc20VerifiableCurrentBalanceOfWalletResponse{
			Error:  &errStr,
			Result: nil,
			Proof:  nil,
		})
		return
	}

	vProof, _, err := verkle.SerializeProof(proof)
	if err != nil {
		errStr := fmt.Sprintf("Failed to serialize proof due to %v", err)
		c.JSON(http.StatusInternalServerError, Brc20VerifiableCurrentBalanceOfWalletResponse{
			Error:  &errStr,
			Result: nil,
			Proof:  nil,
		})
		return
	}
	vProofBytes, err := vProof.MarshalJSON()
	if err != nil {
		errStr := fmt.Sprintf("Failed to marshal the proof to JSON due to %v", err)
		c.JSON(http.StatusInternalServerError, Brc20VerifiableCurrentBalanceOfWalletResponse{
			Error:  &errStr,
			Result: nil,
			Proof:  nil,
		})
		return
	}
	finalproof := base64.StdEncoding.EncodeToString(vProofBytes[:])

	c.JSON(http.StatusOK, Brc20VerifiableCurrentBalanceOfWalletResponse{
		Error:  nil,
		Result: &result,
		Proof:  &finalproof,
	})
}

func GetBlockHeight(c *gin.Context, queue *stateless.Queue) {
	curHeight := queue.LatestHeight()
	c.Data(http.StatusOK, "text/plain", []byte(fmt.Sprintf("%d", curHeight)))
}

func GetLatestStateProof(c *gin.Context, queue *stateless.Queue) {
	if queue.LastStateProof == nil {
		c.JSON(http.StatusOK, Brc20VerifiableLatestStateProofResponse{
			Error:  nil,
			Result: nil,
			Proof:  nil,
		})
		return
	}
	vProof, stateDiff, err := verkle.SerializeProof(queue.LastStateProof)
	if err != nil {
		errStr := fmt.Sprintf("Failed to generate proof due to %v", err)
		c.JSON(http.StatusInternalServerError, Brc20VerifiableLatestStateProofResponse{
			Error:  &errStr,
			Result: nil,
			Proof:  nil,
		})
		return
	}
	vProofBytes, err := vProof.MarshalJSON()
	if err != nil {
		errStr := fmt.Sprintf("Failed to marshal the proof to JSON due to %v", err)
		c.JSON(http.StatusInternalServerError, Brc20VerifiableLatestStateProofResponse{
			Error:  &errStr,
			Result: nil,
			Proof:  nil,
		})
		return
	}

	finalproof := base64.StdEncoding.EncodeToString(vProofBytes[:])

	stateDiffExport := make([]string, 0)
	for _, sd := range stateDiff {
		bytes, err := sd.MarshalJSON()
		if err != nil {
			errStr := fmt.Sprintf("Failed to encode stateDiff due to %v", err)
			c.JSON(http.StatusInternalServerError, Brc20VerifiableLatestStateProofResponse{
				Error:  &errStr,
				Result: nil,
				Proof:  nil,
			})
		}
		str := base64.StdEncoding.EncodeToString(bytes)
		stateDiffExport = append(stateDiffExport, str)
	}

	ordTransfers := queue.Header.OrdTrans

	var ordTransfersJSON []interface{}

	for _, event := range ordTransfers {
		ordTransfersJSON = append(ordTransfersJSON, event)
	}

	res := Brc20VerifiableLatestStateProofResult{
		StateDiff:    stateDiffExport,
		OrdTransfers: ordTransfersJSON,
	}

	c.JSON(http.StatusOK, Brc20VerifiableLatestStateProofResponse{
		Error:  nil,
		Result: &res,
		Proof:  &finalproof,
	})
}

func StartService(queue *stateless.Queue, enableCommittee, enableDebug, enablePprof bool) {
	if !enableDebug {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	r.Use(gin.Recovery(), gin.Logger(), cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"POST", "GET"},
		AllowHeaders:     []string{"*"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	r.Use(metrics.HTTP)

	if enablePprof {
		pprof.Register(r)
	}

	r.GET("/v1/brc20_verifiable/current_balance_of_wallet", func(c *gin.Context) {
		GetCurrentBalanceOfWallet(c, queue)
	})

	r.GET("/v1/brc20_verifiable/block_height", func(c *gin.Context) {
		GetBlockHeight(c, queue)
	})

	r.GET("/healthcheck", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
		})
	})

	if enableCommittee {
		r.GET("/v1/brc20_verifiable/latest_state_proof", func(c *gin.Context) {
			GetLatestStateProof(c, queue)
		})
	}

	// TODO: Medium. Allow user to setup port.
	if err := r.Run(":8080"); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
