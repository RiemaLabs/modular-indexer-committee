package getter

import (
	"github.com/RiemaLabs/modular-indexer-committee/ord"
)

type BRC20Event interface {
	GetEventType() string
}

type Address struct {
	Address ord.Wallet `json:"address"`
}

type BaseEvent struct {
	BlockHeight    uint
	EventType      string `json:"type"`
	Tick           string `json:"tick"`
	InscriptionID  string `json:"inscriptionId"`
	InscriptionNum int32  `json:"inscriptionNumber"`
	OldSatpoint    string `json:"oldSatpoint"`
	NewSatpoint    string `json:"newSatpoint"`
	FromAddress    Address `json:"from"`
	ToAddress      Address `json:"to"`
	Valid          bool   `json:"valid"`
	Msg            string `json:"msg"`
}


func (e *BaseEvent) GetEventType() string {
	return e.EventType
}

type BRC20DeployEvent struct {
	BaseEvent
	Supply       string `json:"supply"`
	LimitPerMint string `json:"limitPerMint"`
	Decimal      int32  `json:"decimal"`
}

type BRC20MintEvent struct {
	BaseEvent
	Amount string `json:"amount"`
}

type BRC20TransferEvent struct {
	BaseEvent
	Amount string `json:"amount"`
}

type BRC20InscribeTransferEvent struct {
	BaseEvent
	Amount string `json:"amount"`
}

type row struct {
	BlockHeight    uint
	EventType      string
	Tick           string
	InscriptionID  string
	InscriptionNum string
	OldSatpoint    string
	NewSatpoint    string
	FromAddress    string
	ToAddress      string
	Valid          bool
	Msg            string
	Supply         string
	LimitPerMint   string
	Decimals        string
	Amount         string
}

type OrdGetter interface {
	GetLatestBlockHeight() (uint, error)
	GetBlockHash(blockHeight uint) (string, error)
	GetOrdTransfers(blockHeight uint) ([]BRC20Event, error)
}
