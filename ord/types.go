package ord

import (
	"sync"
	
	verkle "github.com/ethereum/go-verkle"
	uint256 "github.com/holiman/uint256"
	"github.com/RiemaLabs/indexer-committee/ord/getter"
)

type TripleElement struct {
	Key      [32]byte
	OldValue [32]byte
	NewValue [32]byte
}

type DiffList struct {
	Elements []TripleElement
}

// DiffState stores the difference from next state
type DiffState struct {
	Height uint
	Hash   string

	Diff DiffList
}

type KeyValueMap = map[[32]byte][]byte

type Header struct {
	Root   verkle.VerkleNode
	Height uint
	Hash   string

	KV KeyValueMap
	Temp DiffList
}

type Queue struct {
	Header  Header
	History [BitcoinConfirmations - 1]DiffState
	sync.RWMutex
}

type KVStorage interface {
	Insert(key []byte, value []byte, nodeResolverFn verkle.NodeResolverFn) error

	Get(key []byte, nodeResolverFn verkle.NodeResolverFn) ([]byte, error)

	GetValueOrZero(key []byte) *uint256.Int

	Paging(getter getter.OrdGetter, queryHash bool, nodeResolverFn verkle.NodeResolverFn) error
}
