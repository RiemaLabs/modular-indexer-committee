package stateless

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ethereum/go-verkle"
)

const cachePath = ".cache"
const fileSuffix = ".dat"

func LoadHeader(enableStateRootCache bool, initHeight uint) *Header {
	curHeight := initHeight
	myHeader := Header{
		Root:   verkle.New(),
		Height: curHeight,
		Hash:   "",
		KV:     make(KeyValueMap),
		Diff:   DiffList{},
		TempKV: KeyValueMap{},
	}
	if enableStateRootCache {
		files, err := os.ReadDir(cachePath)
		if err != nil {
			return &myHeader
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
				return &myHeader
			}
			var buffer = bytes.NewBuffer(data)
			log.Println("Start to rebuild verkle tree.")
			storedState, err := Deserialize(buffer, uint(maxHeight), nil)
			if err != nil {
				return &myHeader
			}
			log.Println("End to rebuild verkle tree.")
			return storedState
		}

	}
	return &myHeader
}

func StoreHeader(header *Header, evictHeight uint) error {
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

func StoreKV(header *Header) error {
	fileName := filepath.Join(cachePath, fmt.Sprintf("%d.kv", header.Height))
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()
	keys := header.OrderedKeys()
	for _, k := range keys {
		v := header.KV[k]
		_, err := fmt.Fprintf(file, "0x%v: 0x%v\n", hex.EncodeToString(k[:]), hex.EncodeToString(v[:]))
		if err != nil {
			return err
		}
	}
	return nil
}

func StoreDiff(diff *DiffList, height uint) error {
	fileName := filepath.Join(cachePath, fmt.Sprintf("%d.csv", height))
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()
	fmt.Fprintln(file, "Key,OldValue,NewValue,OldValueExists")
	for _, t := range diff.Elements {
		fmt.Fprintf(file, "0x%x,0x%x,0x%x,%t\n", t.Key, t.OldValue, t.NewValue, t.OldValueExists)
	}
	return nil
}
