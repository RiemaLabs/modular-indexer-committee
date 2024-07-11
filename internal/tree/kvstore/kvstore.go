package kvstore

import (
	"errors"
	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
)

type ByteMap struct {
	DB     *leveldb.DB
	length int
}

func NewByteMap(dbPath string) (*ByteMap, error) {
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, err
	}
	return &ByteMap{
		DB: db,
	}, nil
}

func (bm *ByteMap) Get(key []byte) ([]byte, error) {
	value, err := bm.DB.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("key not found for key: %x", key)
		}
		return nil, err
	}
	return value, nil
}

// Insert adds or updates a key-value pair in the map.
func (bm *ByteMap) Insert(key []byte, value []byte) error {
	err := bm.DB.Put(key, value, nil)
	if err != nil {
		return err
	}
	bm.length++
	return nil
}

func (bm *ByteMap) Delete(key []byte) error {
	err := bm.DB.Delete(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return errors.New("key not found")
		}
		return err
	}

	bm.length--
	return nil
}

// Length returns the number of elements in the map.
func (bm *ByteMap) Length() int {
	return bm.length
}

func (bm *ByteMap) PathClean(key []byte, flushAtDepth byte) error {
	startIndex := int(flushAtDepth)
	if startIndex >= len(key) {
		return errors.New("startIndex >= len(key)")
	}
	for i := startIndex + 1; i <= len(key); i++ {
		_ = bm.Delete(key[:i])
	}
	return nil
}

func (bm *ByteMap) Close() error {
	return bm.DB.Close()
}
