package ord

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"sync"

	verkle "github.com/ethereum/go-verkle"
	uint256 "github.com/holiman/uint256"

	"github.com/RiemaLabs/indexer-committee/ord/getter"
)

func (state State) Copy() State {
	newKV := make(KeyValueMap)
	for k, v := range state.KV {
		newKV[k] = v
	}
	return State{
		Root:   state.Root.Copy(),
		KV:     newKV,
		Height: state.Height,
		Hash:   state.Hash,
	}
}

func (state *State) Insert(key []byte, value []byte, nodeResolverFn verkle.NodeResolverFn) error {
	if len(value) != 32 {
		panic(fmt.Sprintf("The length of value is mismatched. It should be 32, but currently it is: %d", len(value)))
	}
	state.KV[[32]byte(key)] = [32]byte(value)
	err := state.Root.Insert(key, value, nodeResolverFn)
	return err
}

func (state *State) Get(key []byte, nodeResolverFn verkle.NodeResolverFn) ([]byte, error) {
	return state.Root.Get(key, nodeResolverFn)
}

func (state *State) GetUInt256(key []byte) *uint256.Int {
	res := uint256.NewInt(0)
	value, _ := state.Root.Get(key, nodeResolveFn)
	if len(value) == 0 {
		return res
	}
	return res.SetBytes(value)
}

// Historical state neither uploads Checkpoint nor records hash.
func (state *State) HasHash() bool {
	return state.Hash != ""
}

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

func Deserialize(buffer *bytes.Buffer, height uint) (*State, error) {
	var kv KeyValueMap
	decoder := gob.NewDecoder(buffer)
	err := decoder.Decode(&kv)
	if err != nil {
		return nil, err
	}
	root := verkle.New()
	for k, v := range kv {
		err := root.Insert(k[:], v[:], nodeResolveFn)
		if err != nil {
			return nil, nil
		}
	}
	root.Commit()

	state := State{
		Root:   root,
		KV:     kv,
		Height: height,
		Hash:   "",
	}
	return &state, nil
}

// Maintain a queue of states to prepare for the re-org.
// TODO: Use the first state and stateDiffs to represent states.
type StateQueue struct {
	States [BitcoinConfirmations]State
	sync.RWMutex
}

// Build the queue from the start height.
func NewQueues(getter getter.OrdGetter, initState State, queryHash bool, startHeight uint) (*StateQueue, error) {
	var states [BitcoinConfirmations]State
	state := initState
	for i := startHeight; i <= startHeight+BitcoinConfirmations-1; i++ {
		ordTransfer, err := getter.GetOrdTransfers(i)
		if err != nil {
			return nil, err
		}
		state = Exec(state.Copy(), ordTransfer)
		var hash string
		if queryHash {
			hash, err = getter.GetBlockHash(i)
			if err != nil {
				return nil, err
			}
		} else {
			hash = ""
		}
		state.Height = i
		state.Hash = hash
		states[i-startHeight] = state
	}
	queue := StateQueue{
		States: states,
	}
	return &queue, nil
}

func (queue *StateQueue) StartHeight() uint {
	return queue.States[0].Height
}

func (queue *StateQueue) LastestHeight() uint {
	return queue.StartHeight() + uint(len(queue.States))
}

func (queue *StateQueue) LastestState() State {
	return queue.States[len(queue.States)-1]
}

func (queue *StateQueue) State(blockHeight uint) State {
	return queue.States[blockHeight-queue.StartHeight()]
}

// Offer the latest state and pop the oldest state.
func (queue *StateQueue) Offer(element State) {
	for i := 0; i <= len(queue.States)-2; i++ {
		queue.States[i] = queue.States[i+1]
	}
	queue.States[len(queue.States)-1] = element
}

func (queue *StateQueue) Println() {
	log.Println("====", len(queue.States), "====", queue.StartHeight(), "====")
	for _, node := range queue.States {
		log.Print(node.Height, "*")
	}
}

func (queue *StateQueue) Update(getter getter.OrdGetter, initState State, latestHeight uint) error {
	state := initState
	curHeight := state.Height
	for i := curHeight + 1; i <= latestHeight; i++ {
		ordTransfer, err := getter.GetOrdTransfers(i)
		if err != nil {
			return err
		}
		state = Exec(state.Copy(), ordTransfer)
		hash, err := getter.GetBlockHash(i)
		if err != nil {
			return err
		}
		state.Height = i
		state.Hash = hash
		queue.Offer(state)
	}
	return nil
}

// Check if the reorganization happened.
// If so, return the height where the reorganization happened, else, return 0.
func (queue *StateQueue) CheckForReorg(getter getter.OrdGetter) (uint, error) {
	for i := 0; i <= len(queue.States)-1; i++ {
		state := queue.States[i]
		height := state.Height
		hash := state.Hash
		newHash, err := getter.GetBlockHash(height)
		if err != nil {
			return 0, err
		}
		if hash == newHash {
			continue
		} else {
			return height, nil
		}
	}
	return 0, nil
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
