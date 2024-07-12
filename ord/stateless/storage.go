package stateless

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/RiemaLabs/modular-indexer-committee/internal/metrics"
	"github.com/RiemaLabs/modular-indexer-committee/internal/tree"
)

const CachePath = ".cache"
const FileSuffix = ".dat"
const LRUsize = 100000
const FlushDepth = 2
const VerkleDataPath = ".tmpTreeStore"

func LoadHeader(enableStateRootCache bool, initHeight uint) *Header {
	curHeight := initHeight
	myHeader := Header{
		Root:           tree.NewVerkleTreeWithLRU(LRUsize, FlushDepth, VerkleDataPath),
		Height:         curHeight,
		Access:         AccessList{},
		IntermediateKV: KeyValueMap{},
	}
	metrics.CurrentHeight.Set(float64(myHeader.Height))

	if enableStateRootCache {
		directories, err := os.ReadDir(CachePath)
		if err != nil {
			return &myHeader
		}
		// Variables to keep track of the file with the maximum state.height
		var maxHeight int
		var maxDir string

		// Iterate through all files
		for _, dir := range directories {
			if dir.IsDir() && filepath.Ext(dir.Name()) == FileSuffix {
				heightString := strings.TrimSuffix(dir.Name(), FileSuffix)
				height, err := strconv.Atoi(heightString)
				if err == nil && height > maxHeight {
					maxHeight = height
					maxDir = dir.Name()
				}
			}
		}

		if maxDir != "" {
			storedState, err := Deserialize(uint(maxHeight))
			if err != nil {
				return &myHeader
			}
			log.Printf("Recovered from cache at height %d", maxHeight)
			return storedState
		}
	}
	return &myHeader
}

func StoreHeader(header *Header, evictHeight uint) error {
	err := header.Serialize()
	if err != nil {
		return err
	}

	fileName := fmt.Sprintf("%d%s", header.Height, FileSuffix)
	filePath := filepath.Join(CachePath, fileName)
	// err = CopyDir(VerkleDataPath, filePath)
	err = CopyLevelDB(header.Root, VerkleDataPath, filePath)
	log.Printf("Stored header at height %d", header.Height)
	if err != nil {
		return err
	}

	// Delete old files
	directories, err := os.ReadDir(CachePath)
	if err != nil {
		return err
	}
	for _, dir := range directories {
		// Check if the dir has the suffix
		if dir.IsDir() && filepath.Ext(dir.Name()) == FileSuffix {
			heightString := strings.TrimSuffix(dir.Name(), FileSuffix)
			height, err := strconv.Atoi(heightString)
			if err == nil && height < int(evictHeight) {
				err := os.RemoveAll(filepath.Join(CachePath, dir.Name()))
				if err != nil {
					log.Printf("Failed to remove old file: %s, err: %v", dir.Name(), err)
				} else {
					log.Printf("Removed old file: %s", dir.Name())
				}
			}
		}
	}
	return nil
}

func Deserialize(height uint) (*Header, error) {
	origDir := filepath.Join(CachePath, strconv.Itoa(int(height))+FileSuffix)

	// Delete the existing VerkleDataPath
	os.RemoveAll(VerkleDataPath)
	err := CopyDir(origDir, VerkleDataPath)
	if err != nil {
		return nil, fmt.Errorf("error during copying levelDB: %v", err)
	}

	dbTree := tree.NewVerkleTreeWithLRU(LRUsize, FlushDepth, VerkleDataPath)
	if dbTree == nil {
		return nil, fmt.Errorf("error during creating verkle tree")
	}

	// The call of Commit is necessary to refresh the root commit.
	dbTree.VerkleTree.Commit()

	myHeader := Header{
		Root:           dbTree,
		Height:         height,
		Hash:           "",
		Access:         AccessList{},
		IntermediateKV: KeyValueMap{},
	}
	return &myHeader, nil
}

// CopyLevelDB copies a LevelDB database from an open srcDB to a destination path.
func CopyLevelDB(root *tree.VerkleTreeWithLRU, src string, dest string) error {
	// first close the levelDB
	log.Println("Closing the levelDB")
	err := root.KvStore.Close()
	if err != nil {
		return fmt.Errorf("err when closing leveldb: %v", err)
	}
	// copy the levelDB
	log.Println("Copying the levelDB")
	err = CopyDir(src, dest)
	if err != nil {
		return fmt.Errorf("err when copying dir: %v", err)
	}
	// reopen the levelDB
	log.Println("Reopening the levelDB")
	err = root.KvStore.ReOpen(dest)
	if err != nil {
		return fmt.Errorf("err when reopening leveldb: %v", err)
	}
	return nil
}

func CopyDir(src, dest string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		fileInfo, err := os.Stat(srcPath)
		if err != nil {
			return err
		}

		if fileInfo.IsDir() {
			err = os.MkdirAll(destPath, fileInfo.Mode())
			if err != nil {
				return err
			}
			err = CopyDir(srcPath, destPath)
			if err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			err = os.WriteFile(destPath, data, fileInfo.Mode())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func CleanPath(dirPath string) error {
	return os.RemoveAll(dirPath)
}
