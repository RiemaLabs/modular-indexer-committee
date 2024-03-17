package ord

import (
	"bytes"
	"encoding/gob"

	verkle "github.com/ethereum/go-verkle"

	"github.com/RiemaLabs/indexer-committee/ord/getter"
)

func (state *State) Serialize() (*bytes.Buffer, error) {
	// TODO: Use a native database instead of a key-value store for the state management.
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(state.KV)
	if err != nil {
		return nil, err
	}
	return &buffer, nil
}

func Deserialize(buffer *bytes.Buffer, height uint) (*Header, error) {
	var kv KeyValueMap
	decoder := gob.NewDecoder(buffer)
	err := decoder.Decode(&kv)
	if err != nil {
		return nil, err
	}
	root := verkle.New()
	for k, v := range kv {
		err := root.Insert(k[:], v, NodeResolveFn)
		if err != nil {
			return nil, nil
		}
	}

	myHeader := Header{
		Root:   root,
		KV:     kv,
		Height: height,
		Hash:   "",
		Temp: 	DiffList{},
	}
	return &myHeader, nil
}

func SafeInsert(root verkle.VerkleNode, key []byte, value []byte, resolver verkle.NodeResolverFn) error {
	i := root.(*verkle.InternalNode)
	stem := verkle.KeyToStem(key)
	cur_values, err := i.GetValuesAtStem(stem, resolver)
	if err != nil {
		return err
	}
	cur_values[key[verkle.StemSize]] = value
	return i.InsertValuesAtStem(stem, cur_values, resolver)
}
