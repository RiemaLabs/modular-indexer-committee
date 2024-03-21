package apis

import (
	"encoding/base64"
	"net/http"

	"github.com/RiemaLabs/indexer-committee/ord"
	"github.com/RiemaLabs/indexer-committee/ord/stateless"
	verkle "github.com/ethereum/go-verkle"
	"github.com/gin-gonic/gin"
)

func GetAllBalances(queue *stateless.Queue, tick string, pkScript string) ([]byte, []byte, BRC20VerifiableCurrentBalanceResult) {
	queue.Lock()
	defer queue.Unlock()

	var ordPkscript ord.Pkscript = ord.Pkscript(pkScript)
	availKey, overKey, availableBalance, overallBalance := stateless.GetBalances(queue.Header, tick, ordPkscript)
	availableBalanceStr := availableBalance.String()
	overallBalanceStr := overallBalance.String()

	result := BRC20VerifiableCurrentBalanceResult{
		AvailableBalance: availableBalanceStr,
		OverallBalance:   overallBalanceStr,
	}

	return availKey, overKey, result
}

func GetCurrentBalanceOfWallet(c *gin.Context, queue *stateless.Queue) {
	queue.Lock()
	defer queue.Unlock()

	tick := c.DefaultQuery("tick", "")
	wallet := c.DefaultQuery("wallet", "")

	pkScriptKey, pkScript := stateless.GetLatestPkscript(queue.Header, wallet)

	availKey, overKey, result := GetAllBalances(queue, tick, pkScript)

	keys := [][]byte{pkScriptKey, availKey, overKey}

	// Generate proof
	proofOfKeys, _, _, _, _ := verkle.MakeVerkleMultiProof(queue.Header.Root, nil, keys, stateless.NodeResolveFn)
	vProof, _, _ := verkle.SerializeProof(proofOfKeys)
	vProofBytes, _ := vProof.MarshalJSON()
	finalproof := base64.StdEncoding.EncodeToString(vProofBytes[:])

	c.JSON(http.StatusOK, Brc20VerifiableGetCurrentBalanceOfWalletResponse{
		Error:  "None",
		Result: result,
		Proof:  finalproof,
	})
}

func GetCurrentBalanceOfPkscript(c *gin.Context, queue *stateless.Queue) {
	queue.Lock()
	defer queue.Unlock()

	tick := c.DefaultQuery("tick", "")
	pkScript := c.DefaultQuery("pkscript", "")
	availKey, overKey, result := GetAllBalances(queue, tick, pkScript)

	keys := [][]byte{availKey, overKey}
	// Generate proof
	proofOfKeys, _, _, _, _ := verkle.MakeVerkleMultiProof(queue.Header.Root, nil, keys, stateless.NodeResolveFn)
	vProof, _, _ := verkle.SerializeProof(proofOfKeys)
	vProofBytes, _ := vProof.MarshalJSON()
	finalproof := base64.StdEncoding.EncodeToString(vProofBytes[:])

	c.JSON(http.StatusOK, Brc20VerifiableGetCurrentBalanceOfWalletResponse{
		Error:  "None",
		Result: result,
		Proof:  finalproof,
	})
}

func GetBlockHeight(c *gin.Context, queue *stateless.Queue) {
	queue.Lock()
	defer queue.Unlock()

	curHeight := queue.LatestHeight()
	c.JSON(http.StatusOK, gin.H{
		"latestHeight": curHeight,
	})
}

func GetLatestStateProof(c *gin.Context, queue *stateless.Queue) {
	queue.Lock()
	defer queue.Unlock()

	lastIndex := len(queue.History) - 1
	postState := queue.Header.Root
	preState, keys, info := stateless.Rollingback(&queue.History[lastIndex])

	proofOfKeys, _, _, _, _ := verkle.MakeVerkleMultiProof(preState, postState, keys, stateless.NodeResolveFn)
	vProof, _, _ := verkle.SerializeProof(proofOfKeys)
	vProofBytes, _ := vProof.MarshalJSON()
	finalproof := base64.StdEncoding.EncodeToString(vProofBytes[:])

	keysStr := make([]string, len(keys))
	keyExists := make([]bool, len(info))
	preValuesStr := make([]string, len(info))
	postValuesStr := make([]string, len(info))

	for i, elem := range info {
		keysStr[i] = base64.StdEncoding.EncodeToString(elem.Key[:])
		keyExists[i] = elem.OldValueExists
		preValuesStr[i] = base64.StdEncoding.EncodeToString(elem.OldValue[:])
		postValuesStr[i] = base64.StdEncoding.EncodeToString(elem.NewValue[:])
	}

	ordTransfers := queue.Header.OrdTrans

	var ordTransfersJSON []OrdTransferJSON

	for _, ordTransfer := range ordTransfers {
		ordTransfersJSON = append(ordTransfersJSON, OrdTransferJSON{
			ID:            ordTransfer.ID,
			InscriptionID: ordTransfer.InscriptionID,
			NewSatpoint:   ordTransfer.OldSatpoint, // Assuming you want to map OldSatpoint to NewSatpoint
			NewPkscript:   ordTransfer.NewPkscript,
			NewWallet:     ordTransfer.NewWallet,
			SentAsFee:     ordTransfer.SentAsFee,
			Content:       base64.StdEncoding.EncodeToString(ordTransfer.Content),
			ContentType:   ordTransfer.ContentType,
		})
	}

	c.JSON(http.StatusOK, Brc20VerifiableLatestStateProofResponse{
		Keys:       keysStr,
		KeyExists:  keyExists,
		PreValues:  preValuesStr,
		PostValues: postValuesStr,
		Proof:      finalproof,
		OrdTrans:   ordTransfersJSON, // Assuming ordTransfer is correctly typed and can be directly included
	})
}

func StartService(queue *stateless.Queue) {
	r := gin.Default()

	r.GET("/brc20_verifiable_current_balance_of_wallet", func(c *gin.Context) {
		GetCurrentBalanceOfWallet(c, queue)
	})

	r.GET("/brc20_verifiable_current_balance_of_pkscript", func(c *gin.Context) {
		GetCurrentBalanceOfPkscript(c, queue)
	})

	r.GET("/brc20_verifiable_block_height", func(c *gin.Context) {
		GetBlockHeight(c, queue)
	})

	r.GET("/brc20_verifiable_latest_state_proof", func(c *gin.Context) {
		GetLatestStateProof(c, queue)
	})

	r.Run(":8080")
}
