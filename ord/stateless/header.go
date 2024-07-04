package stateless

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-verkle"
	"github.com/holiman/uint256"

	"github.com/RiemaLabs/modular-indexer-committee/internal/metrics"
	"github.com/RiemaLabs/modular-indexer-committee/ord/getter"
)

func (h *Header) insert(key []byte, value []byte) {
	if len(key) != verkle.KeySize {
		panic(fmt.Errorf("the length the key to insert bytes must be %d, current is: %d", verkle.KeySize, len(key)))
	}
	if len(value) != ValueSize {
		panic(fmt.Errorf("the length the value must be %d, current is: %d", ValueSize, len(key)))
	}

	// Get the old value from the verkle tree root.
	oldValue, err := h.Root.Get(key)
	if err != nil {
		panic(err)
	}
	oldValueExists := len(oldValue) > 0

	var keyArray [verkle.KeySize]byte
	copy(keyArray[:], key)

	var oldValueArray [ValueSize]byte
	if oldValueExists {
		copy(oldValueArray[:], oldValue)
	}

	var newValueArray [ValueSize]byte
	copy(newValueArray[:], value)

	// TODO: Medium. Optimize the access to Key-Value to be faster.
	exists := false
	for i, ele := range h.Access.Elements {
		if bytes.Equal(keyArray[:], ele.Key[:]) {
			h.Access.Elements[i].NewValue = newValueArray
			exists = true
			break
		}
	}
	if !exists {
		h.Access.Elements = append(h.Access.Elements, TripleElement{
			Key:            keyArray,
			OldValue:       oldValueArray,
			NewValue:       newValueArray,
			OldValueExists: oldValueExists,
		})
	}

	h.IntermediateKV[[verkle.KeySize]byte(key)] = [ValueSize]byte(value)
}

func (h *Header) get(key []byte) []byte {
	if len(key) != verkle.KeySize {
		panic(fmt.Errorf("the length the key to insert bytes must be %d, current is: %d", verkle.KeySize, len(key)))
	}

	key32 := [verkle.KeySize]byte(key)

	oldValue, err := h.Root.Get(key)
	if err != nil {
		panic(err)
	}
	oldValueExists := len(oldValue) > 0

	var res [ValueSize]byte
	var found bool

	if res, found = h.IntermediateKV[key32]; found {
		// The value has been updated during the execution.
	} else {
		if oldValueExists {
			res = [ValueSize]byte(oldValue)
		} else {
			res = defaultValue()
		}
	}

	// Record access
	exists := false
	for _, ele := range h.Access.Elements {
		if bytes.Equal(key, ele.Key[:]) {
			exists = true
			break
		}
	}
	if !exists {
		h.Access.Elements = append(h.Access.Elements, TripleElement{
			Key:            key32,
			OldValue:       res,
			NewValue:       res,
			OldValueExists: oldValueExists,
		})
	}
	return res[:]
}

func (h *Header) InsertInscriptionID(key []byte, value string) {
	// The first slot contains the first 32 bytes of the InscriptionID
	firstKey := make([]byte, verkle.KeySize)
	copy(firstKey, key)

	valueLen := verkle.LeafValueSize * 2 // 64
	transactionID, err := hex.DecodeString(value[:valueLen])
	if err != nil {
		panic(err)
	}
	h.insert(firstKey, transactionID)

	// The second slot contains the output index of the InscriptionID
	secondKey := make([]byte, verkle.KeySize)
	copy(secondKey, key)
	secondKey[verkle.StemSize] = firstKey[verkle.StemSize] + byte(1)

	outputIndex := value[valueLen+1:]
	outputIndexUint256, err := uint256.FromDecimal(outputIndex)
	if err != nil {
		panic(err)
	}
	h.InsertUInt256(secondKey, outputIndexUint256)
}

func (h *Header) GetInscriptionID(key []byte) string {
	// The first Key
	firstKey := make([]byte, verkle.KeySize)
	copy(firstKey, key)
	transactionIDBytes := h.get(firstKey)
	transactionID := hex.EncodeToString(transactionIDBytes)

	// The second Key
	secondKey := make([]byte, verkle.KeySize)
	copy(secondKey, key)
	secondKey[verkle.StemSize] = firstKey[verkle.StemSize] + byte(1)
	outputIndexUint256 := h.GetUInt256(secondKey)
	outputIndex := outputIndexUint256.Dec()

	return transactionID + "i" + outputIndex
}

func (h *Header) InsertUInt256(key []byte, value *uint256.Int) {
	var dest [ValueSize]byte
	value.WriteToArray32(&dest)
	h.insert(key, dest[:])
}

func (h *Header) GetUInt256(key []byte) *uint256.Int {
	res := uint256.NewInt(0)
	value := h.get(key)
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
		h.insert(newKey, padded[i*ValueSize:(i+1)*ValueSize])
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
		padded = append(padded, h.get(newKey)...)
	}
	res := padded[:len]
	return res
}

func (h *Header) Paging(ordGetter getter.OrdGetter, queryHash bool) error {
	for key, value := range h.IntermediateKV {
		_ = h.Root.Insert(key[:], value[:])
	}

	h.Access = AccessList{}
	h.IntermediateKV = KeyValueMap{}
	// Update height and hash
	h.Height++
	metrics.CurrentHeight.Set(float64(h.Height))
	if queryHash {
		hash, err := ordGetter.GetBlockHash(h.Height)
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

func (h *Header) Serialize() error {
	err := h.Root.Serialization()
	if err != nil {
		return err
	}
	return nil
}
