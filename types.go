package main

import uint256 "github.com/holiman/uint256"

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
