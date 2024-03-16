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

func LoadState(enableStateRootCache bool, initHeight uint) ord.State {
	curHeight := initHeight
	stateRoot := verkle.New()
	state := ord.State{
		Root:   stateRoot,
		KV:     make(ord.KeyValueMap),
		Height: curHeight,
		Hash:   "",
	}

	if enableStateRootCache {
		files, err := os.ReadDir(cachePath)
		if err != nil {
			return state
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
				return state
			}
			var buffer = bytes.NewBuffer(data)
			log.Println("Start to rebuild verkle tree.")
			storedState, err := ord.Deserialize(buffer, uint(maxHeight))
			if err != nil {
				return state
			}
			log.Println("End to rebuild verkle tree.")
			return *storedState
		}
	}
	return state
}

func StoreState(state ord.State, evictHeight uint) error {
	buffer, err := state.Serialize()
	bytes := buffer.Bytes()
	if err != nil {
		return err
	}

	fileName := fmt.Sprintf("%d%s", state.Height, fileSuffix)
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
