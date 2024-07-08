package stateless

import (
	"sync"

	"github.com/RiemaLabs/modular-indexer-committee/ord"
	"github.com/RiemaLabs/modular-indexer-committee/ord/getter"
	verkle "github.com/RiemaLabs/go-verkle"
	tree "github.com/RiemaLabs/modular-indexer-committee/internal/tree"
	uint256 "github.com/holiman/uint256"
)

const ValueSize = 32
const MaxDecimalWidth = 18

type TripleElement struct {
	Key            [verkle.KeySize]byte
	OldValue       [ValueSize]byte
	NewValue       [ValueSize]byte
	OldValueExists bool
}

type AccessList struct {
	Elements []TripleElement
}

// DiffState stores the difference from next state
type DiffState struct {
	Height uint
	Hash   string
	// ipa.CompressedSize
	VerkleCommit [32]byte

	Access AccessList
}

type KeyValueMap = map[[verkle.KeySize]byte][ValueSize]byte

type Header struct {
	// Verkle Tree Root
	Root *tree.VerkleTreeWithLRU

	// The state is after the execution of Block Height.
	Height uint
	// Block Hash.
	Hash string
	// Ord Transfers at Height and Hash.
	OrdTrans []getter.BRC20Event

	// All values being accessed at this height.
	Access AccessList
	// The key-value map during the execution of the block.
	IntermediateKV KeyValueMap

	sync.RWMutex
}

type LightHeader struct {
	// Verkle Tree Root
	Root *tree.VerkleTreeWithLRU
	// The state is after the execution of Block Height.
	Height uint
	// Block Hash.
	Hash string
}

type Queue struct {
	Header         *Header
	History        [ord.BitcoinConfirmations]DiffState
	LastStateProof *verkle.Proof
	sync.RWMutex
}

type KVStorage interface {
	insert(key []byte, value []byte)

	get(key []byte) []byte

	InsertInscriptionID(key []byte, value string)

	GetInscriptionID(key []byte) string

	InsertUInt256(key []byte, value *uint256.Int)

	GetUInt256(key []byte) *uint256.Int

	InsertBytes(key []byte, value []byte)

	GetBytes(key []byte) []byte

	GetHeight() uint
}
