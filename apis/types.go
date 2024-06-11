package apis

import (
	"github.com/RiemaLabs/modular-indexer-committee/ord/getter"
)

type Brc20VerifiableLatestStateProofResult struct {
	StateDiff    []string 		`json:"stateDiff"`
	OrdTransfers []interface{}  `json:"ordTransfers"`
}

// Brc20VerifiableCurrentBalanceOfWallet

type Brc20VerifiableCurrentBalanceOfWalletRequest struct {
	Tick   string `json:"tick"`
	Wallet string `json:"wallet"`
}

type Brc20VerifiableCurrentBalanceOfWalletResult struct {
	AvailableBalance string `json:"availableBalance"`
	OverallBalance   string `json:"overallBalance"`
	Wallet           string `json:"wallet"`
}

type Brc20VerifiableCurrentBalanceOfWalletResponse struct {
	Error  *string                                      `json:"error"`
	Result *Brc20VerifiableCurrentBalanceOfWalletResult `json:"result"`
	Proof  *string                                      `json:"proof"`
}

// Brc20VerifiableLatestStateProof

type Brc20VerifiableLatestStateProofRequest struct {
}

type Brc20VerifiableLatestStateProofResponse struct {
	Error  *string                                `json:"error"`
	Result *Brc20VerifiableLatestStateProofResult `json:"result"`
	Proof  *string                                `json:"proof"`
}

type DeployEventJSON struct {
	EventType      string         `json:"type"`
	Tick           string         `json:"tick"`
	InscriptionID  string         `json:"inscriptionId"`
	InscriptionNum int32          `json:"inscriptionNumber"`
	OldSatpoint    string         `json:"oldSatpoint"`
	NewSatpoint    string         `json:"newSatpoint"`
	FromAddress    getter.Address `json:"from"`
	ToAddress      getter.Address `json:"to"`
	Valid          bool           `json:"valid"`
	Msg            string         `json:"msg"`
	Supply         string         `json:"supply"`
	LimitPerMint   string         `json:"limitPerMint"`
	Decimal        int32          `json:"decimal"`
}

type MintEventJSON struct {
	EventType      string         `json:"type"`
	Tick           string         `json:"tick"`
	InscriptionID  string         `json:"inscriptionId"`
	InscriptionNum int32          `json:"inscriptionNumber"`
	OldSatpoint    string         `json:"oldSatpoint"`
	NewSatpoint    string         `json:"newSatpoint"`
	FromAddress    getter.Address `json:"from"`
	ToAddress      getter.Address `json:"to"`
	Amount         string         `json:"amount"`
}

type TransferEventJSON struct {
	EventType      string         `json:"type"`
	Tick           string         `json:"tick"`
	InscriptionID  string         `json:"inscriptionId"`
	InscriptionNum int32          `json:"inscriptionNumber"`
	OldSatpoint    string         `json:"oldSatpoint"`
	NewSatpoint    string         `json:"newSatpoint"`
	FromAddress    getter.Address `json:"from"`
	ToAddress      getter.Address `json:"to"`
	Amount         string         `json:"amount"`
}

type InscribeTransferEventJSON struct {
	EventType      string         `json:"type"`
	Tick           string         `json:"tick"`
	InscriptionID  string         `json:"inscriptionId"`
	InscriptionNum int32          `json:"inscriptionNumber"`
	OldSatpoint    string         `json:"oldSatpoint"`
	NewSatpoint    string         `json:"newSatpoint"`
	FromAddress    getter.Address `json:"from"`
	ToAddress      getter.Address `json:"to"`
	Amount         string         `json:"amount"`
}
