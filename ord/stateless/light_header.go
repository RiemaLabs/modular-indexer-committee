package stateless

import (
	"fmt"

	"github.com/ethereum/go-verkle"
	uint256 "github.com/holiman/uint256"
)

func (h *LightHeader) insert(key []byte, value []byte, nodeResolverFn verkle.NodeResolverFn) {
	h.Root.Insert(key, value, nodeResolverFn)
}

func (h *LightHeader) get(key []byte, nodeResolverFn verkle.NodeResolverFn) []byte {
	body, err := h.Root.Get(key, nodeResolverFn)
	if err != nil {
		panic(err)
	}
	return body
}

func (h *LightHeader) InsertUInt256(key []byte, value *uint256.Int) {
	var dest [ValueSize]byte
	value.WriteToArray32(&dest)
	h.insert(key, dest[:], nil)
}

func (h *LightHeader) GetUInt256(key []byte) *uint256.Int {
	res := uint256.NewInt(0)
	value := h.get(key, nil)
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
		h.insert(newKey, padded[i*ValueSize:(i+1)*ValueSize], nil)
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
		padded = append(padded, h.get(newKey, nil)...)
	}
	res := padded[:len]
	return res
}

func (h *LightHeader) GetHeight() uint {
	return h.Height
}
