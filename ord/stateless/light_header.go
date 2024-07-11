package stateless

import (
	"encoding/hex"
	"fmt"

	"github.com/RiemaLabs/go-verkle"
	uint256 "github.com/holiman/uint256"
)

func (h *LightHeader) insert(key []byte, value []byte) {
	_ = h.Root.Insert(key, value)
}

func (h *LightHeader) get(key []byte) []byte {
	oldValue, err := h.Root.Get(key)
	if err != nil {
		if err.Error() == "trying to access a node that is missing from the stateless view" {
			// stateless view doesn't include values that first read then write.
			res := defaultValue()
			return res[:]
		} else {
			panic(err)
		}
	}
	return oldValue
}

func (h *LightHeader) InsertInscriptionID(key []byte, value string) {
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

func (h *LightHeader) GetInscriptionID(key []byte) string {
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

func (h *LightHeader) InsertUInt256(key []byte, value *uint256.Int) {
	var dest [ValueSize]byte
	value.WriteToArray32(&dest)
	h.insert(key, dest[:])
}

func (h *LightHeader) GetUInt256(key []byte) *uint256.Int {
	res := uint256.NewInt(0)
	value := h.get(key)
	if len(value) == 0 {
		return res
	}
	return res.SetBytes(value)

}

func (h *LightHeader) InsertBytes(key []byte, value []byte) {
	expectedSize := (verkle.NodeWidth - int(key[verkle.StemSize])) * 32
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

func (h *LightHeader) GetBytes(key []byte) []byte {
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

func (h *LightHeader) GetHeight() uint {
	return h.Height
}
