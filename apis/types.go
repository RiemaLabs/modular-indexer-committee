package apis

import (
	"github.com/RiemaLabs/indexer-committee/ord"
)

type OrdTransferJSON struct {
	ID            uint         `json:"ID"`
	InscriptionID string       `json:"inscriptionID"`
	NewSatpoint   string       `json:"newSatpoint"`
	NewPkScript   ord.PkScript `json:"newPkscript"`
	NewWallet     ord.Wallet   `json:"newWallet"`
	SentAsFee     bool         `json:"sentAsFee"`
	Content       string       `json:"content"`
	ContentType   string       `json:"contentType"`
}

type BRC20VerifiableCurrentBalanceResult struct {
	AvailableBalance string `json:"availableBalance"`
	OverallBalance   string `json:"overallBalance"`
}

type Brc20VerifiableGetCurrentBalanceOfWalletResponse struct {
	Error  string                              `json:"error"`
	Result BRC20VerifiableCurrentBalanceResult `json:"result"`
	Proof  string                              `json:"proof"`
}

type BRC20VerifiableCurrentBalanceOfPkscriptResponse struct {
	Error  string `json:"error"`
	Result BRC20VerifiableCurrentBalanceResult
	Proof  string `json:"proof"`
}

type Brc20VerifiableLatestStateProofResponse struct {
	Keys       []string          `json:"keys"`
	KeyExists  []bool            `json:"keyExists"`
	PreValues  []string          `json:"preValues"`
	PostValues []string          `json:"postValues"`
	Proof      string            `json:"proof"`
	OrdTrans   []OrdTransferJSON `json:"ordTransfer"`
}
