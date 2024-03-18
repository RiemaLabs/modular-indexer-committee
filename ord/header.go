package ord

import (
	"bytes"
	"encoding/gob"

	verkle "github.com/ethereum/go-verkle"
	uint256 "github.com/holiman/uint256"

	"github.com/RiemaLabs/indexer-committee/ord/getter"
)

func NewHeader(getter getter.OrdGetter, initState DiffState) Header {
	myHeader := Header{
		Root:   verkle.New(),
		Height: initState.Height,
		Hash:   initState.Hash,
		KV:     make(KeyValueMap),
		Temp:   DiffList{},
	}

	return myHeader
}

func (h *Header) Insert(key []byte, value []byte, nodeResolverFn verkle.NodeResolverFn) error {
	oldValue, err := h.Get(key, nodeResolverFn)
	if err != nil {
		return err
	}

	var keyArray, oldValueArray, newValueArray [32]byte
	copy(keyArray[:], key)

	if len(oldValue) > 0 {
		copy(oldValueArray[:], oldValue)
	}

	if len(value) > 0 {
		copy(newValueArray[:], value)
	}

	h.Temp.Elements = append(h.Temp.Elements, TripleElement{
		Key:      keyArray,
		OldValue: oldValueArray,
		NewValue: newValueArray,
	})

	return nil
}

func (h *Header) Get(key []byte, nodeResolverFn verkle.NodeResolverFn) ([]byte, error) {
	return h.Root.Get(key, nodeResolverFn)
}

func (h *Header) GetUInt256(key []byte) *uint256.Int {
	res := uint256.NewInt(0)
	value, _ := h.Root.Get(key, NodeResolveFn)
	if len(value) == 0 {
		return res
	}
	return res.SetBytes(value)
}

func (h *Header) Paging(getter getter.OrdGetter, queryHash bool, nodeResolverFn verkle.NodeResolverFn) error {
	for _, elem := range h.Temp.Elements {
		if len(elem.NewValue[:]) == 0 {
			delete(h.KV, elem.Key)
		} else {
			h.KV[elem.Key] = elem.NewValue
		}
		h.Root.Insert(elem.Key[:], elem.NewValue[:], nodeResolverFn)
	}

	h.Temp = DiffList{}
	// Update height and hash
	h.Height++
	if queryHash {
		hash, err := getter.GetBlockHash(h.Height)
		if err != nil {
			return err
		}
		h.Hash = hash
	}
	return nil
}

func (state *Header) Serialize() (*bytes.Buffer, error) {
	// TODO: Use a native database instead of a key-value store for the state management.
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(state.KV)
	if err != nil {
		return nil, err
	}
	return &buffer, nil
}

func Deserialize(buffer *bytes.Buffer, height uint, nodeResolverFn verkle.NodeResolverFn) (*Header, error) {
	var kv KeyValueMap
	decoder := gob.NewDecoder(buffer)
	err := decoder.Decode(&kv)
	if err != nil {
		return nil, err
	}
	root := verkle.New()
	for k, v := range kv {
		err := root.Insert(k[:], v[:], nodeResolverFn)
		if err != nil {
			return nil, nil
		}
	}
	root.Commit()

	myHeader := Header{
		Root:   root,
		KV:     kv,
		Height: height,
		Hash:   "",
		Temp:   DiffList{},
	}
	return &myHeader, nil
}
