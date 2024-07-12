package tree

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/RiemaLabs/modular-indexer-committee/internal/tree/cache"
	"github.com/RiemaLabs/modular-indexer-committee/internal/tree/kvstore"

	verkle "github.com/RiemaLabs/go-verkle"
)

type VerkleTreeWithLRU struct {
	VerkleTree   verkle.VerkleNode
	LRU          *cache.LRUCache
	KvStore      *kvstore.ByteMap
	FlushAtDepth byte
}

func NewVerkleTreeWithLRU(capacity int, flushAtDepth byte, storePath string) *VerkleTreeWithLRU {
	if _, err := os.Stat(storePath); os.IsNotExist(err) {
		log.Println("Store path does not exist, creating a new tree")
		store, err := kvstore.NewByteMap(storePath)
		if err != nil {
			panic(fmt.Errorf("error during creating kvstore: %v", err))
		}
		return &VerkleTreeWithLRU{
			VerkleTree:   verkle.New(),
			LRU:          cache.NewLRUCache(capacity),
			KvStore:      store,
			FlushAtDepth: flushAtDepth, // exclude the node at depth FlushAtDepth， keys in kvStore at least FlushAtDepth+1
		}
	}

	store, err := kvstore.NewByteMap(storePath)
	if err != nil {
		panic(fmt.Errorf("error during creating kvstore: %v", err))
	}

	rootKey, err := hex.DecodeString(strings.Repeat("00", verkle.KeySize+1))
	if err != nil {
		panic(fmt.Errorf("error during decoding root key: %v", err))
	}
	rootSerialzied, err := store.Get(rootKey)
	if err != nil {
		panic(fmt.Errorf("error during getting root node: %v, %s file broken", err, storePath))
	}

	newVerkleTree, err := verkle.CreateInternalNode(rootSerialzied[1:33], rootSerialzied[33:], 0)
	if err != nil {
		panic(fmt.Errorf("error during creating verkle tree from %s: %v", err, storePath))
	}

	return &VerkleTreeWithLRU{
		VerkleTree:   newVerkleTree,
		LRU:          cache.NewLRUCache(capacity),
		KvStore:      store,
		FlushAtDepth: flushAtDepth, // exclude the node at depth FlushAtDepth， keys in kvStore at least FlusgAtDepth+1
	}
}

func (v *VerkleTreeWithLRU) Insert(key, value []byte) error {
	if err := v.VerkleTree.Insert(key, value, v.KvStore.Get); err != nil {
		return err
	}
	if err := v.pagingInsideLRU(key); err != nil {
		return fmt.Errorf("error during paging when inserting: %v", err)
	}

	return nil
}

func (v *VerkleTreeWithLRU) Get(key []byte) ([]byte, error) {
	values, err := v.VerkleTree.Get(key, v.KvStore.Get)
	if err != nil {
		return nil, err
	}
	if err := v.pagingInsideLRU(key); err != nil {
		return nil, fmt.Errorf("error during paging when getting: %v", err)
	}

	return values, nil
}

func (v *VerkleTreeWithLRU) pagingInsideLRU(key []byte) error {
	// process the path of moved out keys in kvram
	v.KvStore.PathClean(key, v.FlushAtDepth)
	// insert the key in LRU
	evictedValue, evicted := v.LRU.Insert(key)
	if evicted {
		// follow evicted key path in verkle tree, if exists, then flush everything **under** internal node into kvram
		// 4 steps: 1) tree.GetNode 2) tree.BatchSerialize 3) tree.HashNodeFromInternal 4) kv.insert
		flushStartNode, startPath, _ := v.VerkleTree.(*verkle.InternalNode).GetInternalNode(evictedValue, v.FlushAtDepth) // flushStartNode is not flushed
		if flushStartNode == nil {
			return nil
		}
		if internalNode, ok := flushStartNode.(*verkle.InternalNode); ok {
			serializedNodes, err := internalNode.BatchSerialize() // not flushed yet at this step
			if err != nil {
				return fmt.Errorf("error during batch serialization: %v", err)
			}
			internalNode.HashNodeFromInternal() // flush under internalNode
			// Insert the serialized nodes into kvStore
			for _, node := range serializedNodes[1:] { // exclude itself
				v.KvStore.Insert(append(startPath, node.Path...), node.SerializedBytes)
			}
		}
	}
	return nil
}

func (v *VerkleTreeWithLRU) Serialization() error {
	serializedNodes, err := v.VerkleTree.(*verkle.InternalNode).BatchSerialize()
	if err != nil {
		return fmt.Errorf("error during batch serialization: %v", err)
	}
	v.VerkleTree.(*verkle.InternalNode).HashNodeFromInternal() // flush under root
	// Insert the serialized nodes into kvStore
	for _, node := range serializedNodes[1:] { // exclude root
		v.KvStore.Insert(node.Path, node.SerializedBytes)
	}
	// store the root node
	rootKey, err := hex.DecodeString(strings.Repeat("00", verkle.KeySize+1))
	if err != nil {
		return fmt.Errorf("error during decoding root key: %v", err)
	}
	v.KvStore.Insert(rootKey, serializedNodes[0].SerializedBytes)
	return nil
}

func (v *VerkleTreeWithLRU) Close() error {
	return v.KvStore.Close()
}

func (v *VerkleTreeWithLRU) ReOpen(path string) error {
	return v.KvStore.ReOpen(path)
}
