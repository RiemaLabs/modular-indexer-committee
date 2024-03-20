package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/RiemaLabs/indexer-committee/ord/stateless"
)

func printCache() {
	header := stateless.LoadHeader(true, stateless.BRC20StartHeight)
	fileName := "__debug_kv"
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer file.Close()
	keys := header.OrderedKeys()
	for _, k := range keys {
		v := header.KV[k]
		_, err := fmt.Fprintf(file, "0x%v: 0x%v\n", hex.EncodeToString(k[:]), hex.EncodeToString(v[:]))
		if err != nil {
			log.Fatalf("Error writing to file: %v", err)
		}
	}
}
