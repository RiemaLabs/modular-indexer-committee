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

type Brc20VerifiableLatestStateProofResult struct {
	StateDiff    []string          `json:"stateDiff"`
	OrdTransfers []OrdTransferJSON `json:"ordTransfers"`
}

// Brc20VerifiableCurrentBalanceOfWallet

type Brc20VerifiableCurrentBalanceOfWalletRequest struct {
	Tick   string `json:"tick"`
	Wallet string `json:"wallet"`
}

type Brc20VerifiableCurrentBalanceOfWalletResult struct {
	AvailableBalance string `json:"availableBalance"`
	OverallBalance   string `json:"overallBalance"`
	Pkscript         string `json:"pkscript"`
}

type Brc20VerifiableCurrentBalanceOfWalletResponse struct {
	Error  *string                                      `json:"error"`
	Result *Brc20VerifiableCurrentBalanceOfWalletResult `json:"result"`
	Proof  *string                                      `json:"proof"`
}

// Brc20VerifiableCurrentBalanceOfPkscript

type Brc20VerifiableCurrentBalanceOfPkscriptRequest struct {
	Tick     string `json:"tick"`
	Pkscript string `json:"pkscript"`
}

type Brc20VerifiableCurrentBalanceOfPkscriptResult struct {
	AvailableBalance string `json:"availableBalance"`
	OverallBalance   string `json:"overallBalance"`
}

type Brc20VerifiableCurrentBalanceOfPkscriptResponse struct {
	Error  *string                                        `json:"error"`
	Result *Brc20VerifiableCurrentBalanceOfPkscriptResult `json:"result"`
	Proof  *string                                        `json:"proof"`
}

// Brc20VerifiableLatestStateProof

type Brc20VerifiableLatestStateProofRequest struct {
}

type Brc20VerifiableLatestStateProofResponse struct {
	Error  *string                                `json:"error"`
	Result *Brc20VerifiableLatestStateProofResult `json:"result"`
	Proof  *string                                `json:"proof"`
}
