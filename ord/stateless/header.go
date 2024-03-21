package stateless

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"sort"

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

func (h *Header) insert(key []byte, value []byte, nodeResolverFn verkle.NodeResolverFn) {
	if len(key) != verkle.KeySize {
		panic(fmt.Errorf("the length the key to insert bytes must be %d, current is: %d", verkle.KeySize, len(key)))
	}
	oldValue := h.get(key, nodeResolverFn)
	var keyArray [verkle.KeySize]byte
	var oldValueArray, newValueArray [ValueSize]byte
	copy(keyArray[:], key)

	if len(oldValue) > 0 {
		copy(oldValueArray[:], oldValue)
	}

	if len(value) > 0 {
		copy(newValueArray[:], value)
	}

	oldExists := true
	if oldValue == nil {
		oldExists = false
	}

	if len(value) != ValueSize {
		panic(fmt.Errorf("the length of the value must be: %d, current is: %d", len(value), ValueSize))
	}

	h.Temp.Elements = append(h.Temp.Elements, TripleElement{
		Key:            keyArray,
		OldValue:       oldValueArray,
		NewValue:       newValueArray,
		OldValueExists: oldExists,
	})
}

func (h *Header) get(key []byte, nodeResolverFn verkle.NodeResolverFn) []byte {
	if len(key) != verkle.KeySize {
		panic(fmt.Errorf("the length the key to insert bytes must be %d, current is: %d", verkle.KeySize, len(key)))
	}
	bytes, err := h.Root.Get(key, nodeResolverFn)
	if err != nil {
		panic(err)
	}
	return bytes
}

func (h *Header) InsertUInt256(key []byte, value *uint256.Int) {
	var dest [ValueSize]byte
	value.WriteToArray32(&dest)
	h.insert(key, dest[:], NodeResolveFn)
}

func (h *Header) GetUInt256(key []byte) *uint256.Int {
	res := uint256.NewInt(0)
	value := h.get(key, NodeResolveFn)
	if len(value) == 0 {
		return res
	}
	return res.SetBytes(value)
}

func (h *Header) InsertBytes(key []byte, value []byte) {
	expectedSize := (verkle.NodeWidth - int(key[verkle.StemSize])) * ValueSize
	if len(value) > expectedSize {
		panic(fmt.Errorf("the max length of the byte is: %d at key %s, current is: %d", expectedSize, key, len(value)))
	}
	// The first slot is the number of required slots to store the byte.
	newKey := make([]byte, verkle.KeySize)
	copy(newKey, key)

	len := len(value)
	requiredSlots := (len + ValueSize - 1) / ValueSize
	h.InsertUInt256(newKey, uint256.NewInt(uint64(len)))

	totalLen := requiredSlots * ValueSize
	padded := make([]byte, totalLen)
	copy(padded, value)

	for i := range requiredSlots {
		newKey[verkle.StemSize] = key[verkle.StemSize] + byte(i+1)
		h.insert(newKey, padded[i*ValueSize:(i+1)*ValueSize], NodeResolveFn)
	}
}

func (h *Header) GetBytes(key []byte) []byte {
	newKey := make([]byte, verkle.KeySize)
	copy(newKey, key)

	len := h.GetUInt256(newKey).Uint64()
	if len == 0 {
		return make([]byte, 0)
	}
	requiredSlots := (len + ValueSize - 1) / ValueSize

	padded := make([]byte, 0)
	for i := range requiredSlots {
		newKey[verkle.StemSize] = key[verkle.StemSize] + byte(i+1)
		padded = append(padded, h.get(newKey, NodeResolveFn)...)
	}
	res := padded[:len]
	return res
}

// h.Height ++
func (h *Header) Paging(getter getter.OrdGetter, queryHash bool, nodeResolverFn verkle.NodeResolverFn) error {
	for _, elem := range h.Temp.Elements {
		h.KV[elem.Key] = elem.NewValue
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

func (h *Header) GetHeight() uint {
	return h.Height
}

func (h *Header) Serialize() (*bytes.Buffer, error) {
	// TODO: Medium. Use a native database instead of a key-value store for the state management.
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(h.KV)
	if err != nil {
		return nil, err
	}
	return &buffer, nil
}

func (h *Header) OrderedKeys() [][verkle.KeySize]byte {
	keys := make([][verkle.KeySize]byte, 0, len(h.KV))
	for key := range h.KV {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return string(keys[i][:]) < string(keys[j][:])
	})
	return keys
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
