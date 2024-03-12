package main

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
}

// The first block height of the brc-20 protocol.
var BRC20StartHeight uint = 779832

// The number of confirmations to be considered immutable.
// Keep the same with OPI.
// TODO: Find the original code of OPI.
var BitcoinConfirmations uint = 10

// Version Control
var Version string

var GlobalConfig Config
