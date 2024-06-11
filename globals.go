package main

type Config struct {
	Database struct {
		EventUrl     string `json:"eventUrl"`
		HashUrl	   string `json:"hashUrl"`
	} `json:"database"`
	Report struct {
		Method  string `json:"method"`
		Timeout int    `json:"timeout"`
		S3      struct {
			Bucket    string `json:"bucket"`
			Region    string `json:"region"`
			AccessKey string `json:"accessKey"`
			SecretKey string `json:"secretKey"`
		} `json:"s3"`
		Da struct {
			Network     string `json:"network"`
			NamespaceID string `json:"namespaceID"`
			GasCoupon   string `json:"gasCoupon"`
			PrivateKey  string `json:"privateKey"`
		} `json:"da"`
	} `json:"report"`
	Service struct {
		Name         string `json:"name"`
		URL          string `json:"url"`
		MetaProtocol string `json:"metaProtocol"`
	} `json:"service"`
}

var GlobalConfig Config
