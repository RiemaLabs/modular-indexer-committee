package apis

import (
	"encoding/base64"
	"fmt"
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

func GetAllBalances(queue *stateless.Queue, tick string, pkScript string) ([]byte, []byte, Brc20VerifiableCurrentBalanceOfPkscriptResult) {
	var ordPkscript ord.Pkscript = ord.Pkscript(pkScript)
	availKey, overKey, availableBalance, overallBalance := stateless.GetBalances(queue.Header, tick, ordPkscript)
	availableBalanceStr := availableBalance.String()
	overallBalanceStr := overallBalance.String()

	result := Brc20VerifiableCurrentBalanceOfPkscriptResult{
		AvailableBalance: availableBalanceStr,
		OverallBalance:   overallBalanceStr,
	}

	return availKey, overKey, result
}

func GetCurrentBalanceOfWallet(c *gin.Context, queue *stateless.Queue) {
	tick := c.DefaultQuery("tick", "")
	wallet := c.DefaultQuery("wallet", "")

	_, pkScript := stateless.GetLatestPkscript(queue.Header, wallet)

	availKey, overKey, result := GetAllBalances(queue, tick, pkScript)

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

	resultWallet := Brc20VerifiableCurrentBalanceOfWalletResult{
		AvailableBalance: result.AvailableBalance,
		OverallBalance:   result.OverallBalance,
		Pkscript:         pkScript,
	}

	c.JSON(http.StatusOK, Brc20VerifiableCurrentBalanceOfWalletResponse{
		Error:  nil,
		Result: &resultWallet,
		Proof:  &finalproof,
	})
}

func GetCurrentBalanceOfPkscript(c *gin.Context, queue *stateless.Queue) {
	tick := c.DefaultQuery("tick", "")
	pkScript := c.DefaultQuery("pkscript", "")
	availKey, overKey, result := GetAllBalances(queue, tick, pkScript)

	keys := [][]byte{availKey, overKey}
	// Generate proof
	proofOfKeys, _, _, _, err := verkle.MakeVerkleMultiProof(queue.Header.Root, nil, keys, stateless.NodeResolveFn)
	if err != nil {
		errStr := fmt.Sprintf("Failed to generate proof due to %v", err)
		c.JSON(http.StatusInternalServerError, Brc20VerifiableCurrentBalanceOfPkscriptResponse{
			Error:  &errStr,
			Result: nil,
			Proof:  nil,
		})
		return
	}
	vProof, _, err := verkle.SerializeProof(proofOfKeys)
	if err != nil {
		errStr := fmt.Sprintf("Failed to serialize proof due to %v", err)
		c.JSON(http.StatusInternalServerError, Brc20VerifiableCurrentBalanceOfPkscriptResponse{
			Error:  &errStr,
			Result: nil,
			Proof:  nil,
		})
		return
	}

	vProofBytes, err := vProof.MarshalJSON()
	if err != nil {
		errStr := fmt.Sprintf("Failed to marshal the proof to JSON due to %v", err)
		c.JSON(http.StatusInternalServerError, Brc20VerifiableCurrentBalanceOfPkscriptResponse{
			Error:  &errStr,
			Result: nil,
			Proof:  nil,
		})
		return
	}
	finalproof := base64.StdEncoding.EncodeToString(vProofBytes[:])

	c.JSON(http.StatusOK, Brc20VerifiableCurrentBalanceOfPkscriptResponse{
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

	var ordTransfersJSON []OrdTransferJSON

	for _, ordTransfer := range ordTransfers {
		ordTransfersJSON = append(ordTransfersJSON, OrdTransferJSON{
			ID:            ordTransfer.ID,
			InscriptionID: ordTransfer.InscriptionID,
			OldSatpoint:   ordTransfer.OldSatpoint,
			NewSatpoint:   ordTransfer.NewSatpoint,
			NewPkscript:   ordTransfer.NewPkscript,
			NewWallet:     ordTransfer.NewWallet,
			SentAsFee:     ordTransfer.SentAsFee,
			Content:       base64.StdEncoding.EncodeToString(ordTransfer.Content),
			ContentType:   ordTransfer.ContentType,
		})
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

	r.GET("/v1/brc20_verifiable/current_balance_of_pkscript", func(c *gin.Context) {
		GetCurrentBalanceOfPkscript(c, queue)
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
	r.Run(":8080")
}
