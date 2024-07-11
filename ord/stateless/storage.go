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
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/opt"
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
	err = CopyLevelDB(header.Root.KvStore.DB, filePath)
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
func CopyLevelDB(srcDB *leveldb.DB, dest string) error {
	// Create a snapshot of the source DB.
	snapshot, err := srcDB.GetSnapshot()
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}
	defer snapshot.Release()

	// Create the destination directory if it doesn't exist.
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Open the destination LevelDB.
	destDB, err := leveldb.OpenFile(dest, &opt.Options{ErrorIfExist: false})
	if err != nil {
		if errors.IsCorrupted(err) {
			destDB, err = leveldb.RecoverFile(dest, nil)
			if err != nil {
				return fmt.Errorf("failed to recover destination DB: %w", err)
			}
		} else {
			return fmt.Errorf("failed to open destination DB: %w", err)
		}
	}
	defer destDB.Close()

	// Begin a batch write.
	batch := new(leveldb.Batch)
	iter := snapshot.NewIterator(nil, nil)
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		batch.Put(key, value)
	}
	iter.Release()

	if err := iter.Error(); err != nil {
		return fmt.Errorf("iteration error: %w", err)
	}

	// Write the batch to the destination DB.
	if err := destDB.Write(batch, nil); err != nil {
		return fmt.Errorf("failed to write batch to destination DB: %w", err)
	}

	return nil
}

func CopyDir(src, dest string) error {
	// Open the source LevelDB.
	srcDB, err := leveldb.OpenFile(src, nil)
	if err != nil {
		return fmt.Errorf("failed to open source DB: %w", err)
	}
	defer srcDB.Close()

	// Create a snapshot of the source DB.
	snapshot, err := srcDB.GetSnapshot()
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}
	defer snapshot.Release()

	// Create the destination directory if it doesn't exist.
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Open the destination LevelDB.
	destDB, err := leveldb.OpenFile(dest, &opt.Options{ErrorIfExist: true})
	if err != nil {
		if errors.IsCorrupted(err) {
			destDB, err = leveldb.RecoverFile(dest, nil)
			if err != nil {
				return fmt.Errorf("failed to recover destination DB: %w", err)
			}
		} else {
			return fmt.Errorf("failed to open destination DB: %w", err)
		}
	}
	defer destDB.Close()

	// Begin a batch write.
	batch := new(leveldb.Batch)
	iter := snapshot.NewIterator(nil, nil)
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		batch.Put(key, value)
	}
	iter.Release()

	if err := iter.Error(); err != nil {
		return fmt.Errorf("iteration error: %w", err)
	}

	// Write the batch to the destination DB.
	if err := destDB.Write(batch, nil); err != nil {
		return fmt.Errorf("failed to write batch to destination DB: %w", err)
	}

	return nil
}

func CleanPath(dirPath string) error {
	return os.RemoveAll(dirPath)
}
