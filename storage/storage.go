package storage

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/RiemaLabs/indexer-committee/ord"
	"github.com/ethereum/go-verkle"
)

const cachePath = ".cache"
const fileSuffix = ".dat"

func LoadHeader(enableStateRootCache bool, initHeight uint) ord.Header {
	curHeight := initHeight
	myHeader := ord.Header{
		Root:   verkle.New(),
		Height: curHeight,
		Hash:   "",
		KV:     make(ord.KeyValueMap),
		Temp:   ord.DiffList{},
	}
	if enableStateRootCache {
		files, err := os.ReadDir(cachePath)
		if err != nil {
			return myHeader
		}
		// Variables to keep track of the file with the maximum state.height
		var maxHeight int
		var maxFile string

		// Iterate through all files
		for _, file := range files {
			// Check if the file has the suffix
			if filepath.Ext(file.Name()) == fileSuffix {
				heightString := strings.TrimSuffix(file.Name(), fileSuffix)
				height, err := strconv.Atoi(heightString)
				if err == nil && height > maxHeight {
					// Update the maximum state.height and corresponding file name
					maxHeight = height
					maxFile = file.Name()
				}
			}
		}

		if maxFile != "" {
			data, err := os.ReadFile(filepath.Join(cachePath, maxFile))
			if err != nil {
				return myHeader
			}
			var buffer = bytes.NewBuffer(data)
			log.Println("Start to rebuild verkle tree.")
			storedState, err := ord.Deserialize(buffer, uint(maxHeight), nil)
			if err != nil {
				return myHeader
			}
			log.Println("End to rebuild verkle tree.")
			return *storedState
		}

	}
	return myHeader
}

func StoreState(header ord.Header, evictHeight uint) error {
	buffer, err := header.Serialize()
	bytes := buffer.Bytes()
	if err != nil {
		return err
	}

	fileName := fmt.Sprintf("%d%s", header.Height, fileSuffix)
	filePath := filepath.Join(cachePath, fileName)
	err = os.WriteFile(filePath, bytes, 0666)
	if err != nil {
		return err
	}

	// Delete old files
	files, err := os.ReadDir(cachePath)
	if err != nil {
		return err
	}
	for _, file := range files {
		// Check if the file has the suffix
		if filepath.Ext(file.Name()) == fileSuffix {
			heightString := strings.TrimSuffix(file.Name(), fileSuffix)
			height, err := strconv.Atoi(heightString)
			if err == nil && height < int(evictHeight) {
				err := os.Remove(filepath.Join(cachePath, file.Name()))
				if err != nil {
					log.Printf("Failed to remove old file: %s, err: %v", file.Name(), err)
				}
			}
		}
	}
	return nil
}
