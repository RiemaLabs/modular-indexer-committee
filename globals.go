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
	Service struct {
		URL          string `json:"url"`
		Name         string `json:"name"`
		MetaProtocol string `json:"metaProtocol"`
	} `json:"service"`
}

// Version Control
var Version string

var GlobalConfig Config
