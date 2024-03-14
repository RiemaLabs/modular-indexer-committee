package main

import (
	uint256 "github.com/holiman/uint256"
	"gorm.io/gorm"
)

type Config struct {
	Database struct {
		Host     string `json:"host"`
		User     string `json:"user"`
		Password string `json:"password"`
		DBname   string `json:"dbname"`
		Port     string `json:"port"`
	} `json:"database"`
	Report struct {
		Method   string `json:"method"`
		UniqueID string `json:"uniqueID"`
		S3       struct {
			Bucket    string `json:"bucket"`
			AccessKey string `json:"accessKey"`
		} `json:"s3"`
		Da struct{} `json:"da"`
	} `json:"report"`
	BitcoinRPC struct {
		URL      string `json:"url"`
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"bitcoinRPC"`
	Service struct {
		URL          string `json:"url"`
		Name         string `json:"name"`
		MetaProtocol string `json:"metaProtocol"`
	} `json:"service"`
}

type OrdTransfer struct {
	ID            uint
	InscriptionID string
	OldSatpoint   string
	NewPkscript   string
	NewWallet     string
	SentAsFee     bool
	Content       []byte
	ContentType   string
}

type BRC20Tickers struct {
	Tick            string
	RemainingSupply string
	LimitPerMint    string
	Decimals        string
}

type Event struct {
	SourcePkScript string
	SourceWallet   string
	Tick           string
	Amount         *uint256.Int
	UsingTxId      string
}

type Checkpoint struct {
	URL          string
	Name         string
	Version      string
	MetaProtocol string
	Height       string
	Hash         string
	Commitment   string
}

type StateDiff struct {
	Key   string
	Value []byte
}

type BRC20HistoricBalances struct {
	gorm.Model
	ID               uint   `gorm:"primary_key;auto_increment"`
	Pkscript         string `gorm:"type:text;not null"`
	Wallet           string `gorm:"type:text;not null"`
	Tick             string `gorm:"type:varchar(4);not null"`
	OverallBalance   string `gorm:"type:numeric(40);not null"`
	AvailableBalance string `gorm:"type:numeric(40);not null"`
	BlockHeight      int    `gorm:"type:int;not null"`
	EventID          int64  `gorm:"type:bigint;not null"`
}

type BitcoinGetter interface {
	GetLatestBlockHeight() (uint, error)
	GetBlockHash(blockHeight uint) (string, error)
	GetOrdTransfers(blockHeight uint) ([]OrdTransfer, error)
}
