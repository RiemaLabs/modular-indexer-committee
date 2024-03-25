package apis

import (
	"github.com/RiemaLabs/modular-indexer-committee/ord"
)

type OrdTransferJSON struct {
	ID            uint         `json:"ID"`
	InscriptionID string       `json:"inscriptionID"`
	NewSatpoint   string       `json:"newSatpoint"`
	NewPkscript   ord.Pkscript `json:"newPkscript"`
	NewWallet     ord.Wallet   `json:"newWallet"`
	SentAsFee     bool         `json:"sentAsFee"`
	Content       string       `json:"content"`
	ContentType   string       `json:"contentType"`
}

type Brc20VerifiableCurrentBalanceResult struct {
	AvailableBalance string `json:"availableBalance"`
	OverallBalance   string `json:"overallBalance"`
}

type Brc20VerifiableLatestStateProofResult struct {
	Keys         []string          `json:"keys"`
	KeyExists    []bool            `json:"keyExists"`
	PreValues    []string          `json:"preValues"`
	PostValues   []string          `json:"postValues"`
	OrdTransfers []OrdTransferJSON `json:"ordTransfer"`
}

// Brc20VerifiableCurrentBalanceOfWallet

type Brc20VerifiableCurrentBalanceOfWalletRequest struct {
	Tick   string `json:"tick"`
	Wallet string `json:"wallet"`
}

type Brc20VerifiableCurrentBalanceOfWalletResponse struct {
	Error  *string                              `json:"error"`
	Result *Brc20VerifiableCurrentBalanceResult `json:"result"`
	Proof  *string                              `json:"proof"`
}

// Brc20VerifiableCurrentBalanceOfPkscript

type Brc20VerifiableCurrentBalanceOfPkscriptRequest struct {
	Tick     string `json:"tick"`
	Pkscript string `json:"pkscript"`
}

type Brc20VerifiableCurrentBalanceOfPkscriptResponse struct {
	Error  *string `json:"error"`
	Result *Brc20VerifiableCurrentBalanceResult
	Proof  *string `json:"proof"`
}

// Brc20VerifiableLatestStateProof

type Brc20VerifiableLatestStateProofRequest struct {
}

type Brc20VerifiableLatestStateProofResponse struct {
	Error  *string `json:"error"`
	Result *Brc20VerifiableLatestStateProofResult
	Proof  *string `json:"proof"`
}
