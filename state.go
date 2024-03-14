package main

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	verkle "github.com/ethereum/go-verkle"
	uint256 "github.com/holiman/uint256"
)

const Suffix = ".dat"

type KeyValueMap = map[[32]byte][]byte

type State struct {
	root   verkle.VerkleNode
	kv     KeyValueMap
	height uint
	hash   string
}

func (state State) Copy() State {
	newKV := make(KeyValueMap)
	for k, v := range state.kv {
		newKV[k] = v
	}
	return State{
		root:   state.root.Copy(),
		kv:     newKV,
		height: state.height,
		hash:   state.hash,
	}
}

func (state *State) Insert(key []byte, value []byte, nodeResolverFn verkle.NodeResolverFn) error {
	state.kv[[32]byte(key)] = value
	err := state.root.Insert(key, value, nodeResolverFn)
	return err
}

func (state *State) Get(key []byte, nodeResolverFn verkle.NodeResolverFn) ([]byte, error) {
	return state.root.Get(key, nodeResolverFn)
}

func (state *State) GetValueOrZero(key []byte) *uint256.Int {
	res := uint256.NewInt(0)
	value, _ := state.root.Get(key, nodeResolveFn)
	if len(value) == 0 {
		return res
	}
	return res.SetBytes(value)
}

// Historical state neither uploads Checkpoint nor records hash.
func (state *State) HasHash() bool {
	return state.hash != ""
}

func (state *State) Checkpoint(config Config) Checkpoint {
	blockHeight := strconv.FormatUint(uint64(state.height), 10)
	blockHash := state.hash
	bytes := state.root.Commit().Bytes()
	commitment := base64.StdEncoding.EncodeToString(bytes[:])
	content := Checkpoint{
		URL:          config.Service.URL,
		Name:         config.Service.Name,
		Version:      Version,
		MetaProtocol: config.Service.MetaProtocol,
		Height:       blockHeight,
		Hash:         blockHash,
		Commitment:   commitment,
	}
	return content
}

func (state *State) SerializeToFile(path string) error {
	// TODO: Using a native database instead of a key-value store for state management.
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(state.kv)
	if err != nil {
		return err
	}

	fileName := fmt.Sprintf("%d%s", state.height, Suffix)
	filePath := filepath.Join(path, fileName)
	err = os.WriteFile(filePath, buffer.Bytes(), 0666)
	if err != nil {
		return err
	}
	return nil
}

func DeserializeLatestState(path string) (*State, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	// Variables to keep track of the file with the maximum state.height
	var maxHeight int
	var maxFile string

	// Iterate through all files
	for _, file := range files {
		// Check if the file has the suffix
		if filepath.Ext(file.Name()) == Suffix {
			heightString := strings.TrimSuffix(file.Name(), Suffix)
			height, err := strconv.Atoi(heightString)
			if err == nil && height > maxHeight {
				// Update the maximum state.height and corresponding file name
				maxHeight = height
				maxFile = file.Name()
			}
		}
	}
	if maxFile != "" {
		data, err := os.ReadFile(filepath.Join(path, maxFile))
		if err != nil {
			return nil, err
		}
		var buffer bytes.Buffer
		var kv KeyValueMap
		buffer = *bytes.NewBuffer(data)
		decoder := gob.NewDecoder(&buffer)
		err = decoder.Decode(&kv)
		if err != nil {
			return nil, err
		}
		root := verkle.New()
		for k, v := range kv {
			root.Insert(k[:], v, nodeResolveFn)
		}
		state := State{
			root:   root,
			kv:     kv,
			height: uint(maxHeight),
			hash:   "",
		}
		return &state, nil
	} else {
		return nil, nil
	}
}

// Maintain a queue of states to prepare for the re-org.
// TODO: Use the first state and stateDiffs to represent states.
type StateQueue struct {
	states [BitcoinConfirmations]State
	sync.RWMutex
}

// Build the queue from the start height.
func NewQueues(getter BitcoinGetter, initState State, queryHash bool, startHeight uint) (*StateQueue, error) {
	var states [BitcoinConfirmations]State
	state := initState
	for i := startHeight; i <= startHeight+BitcoinConfirmations-1; i++ {
		ordTransfer, err := getter.GetOrdTransfers(i)
		if err != nil {
			return nil, err
		}
		state = processOrdTransfer(state.Copy(), ordTransfer)
		var hash string
		if queryHash {
			hash, err = getter.GetBlockHash(i)
			if err != nil {
				return nil, err
			}
		} else {
			hash = ""
		}
		state.height = i
		state.hash = hash
		states[i-startHeight] = state
	}
	queue := StateQueue{
		states: states,
	}
	return &queue, nil
}

func (queue *StateQueue) StartHeight() uint {
	return queue.states[0].height
}

func (queue *StateQueue) LastestHeight() uint {
	return queue.StartHeight() + uint(len(queue.states))
}

func (queue *StateQueue) LastestState() State {
	return queue.states[len(queue.states)-1]
}

func (queue *StateQueue) State(blockHeight uint) State {
	return queue.states[blockHeight-queue.StartHeight()]
}

// Offer the latest state and pop the oldest state.
func (queue *StateQueue) Offer(element State) {
	for i := 0; i <= len(queue.states)-2; i++ {
		queue.states[i] = queue.states[i+1]
	}
	queue.states[len(queue.states)-1] = element
}

func (queue *StateQueue) Println() {
	log.Println("====", len(queue.states), "====", queue.StartHeight(), "====")
	for _, node := range queue.states {
		log.Print(node.height, "*")
	}
}

func (queue *StateQueue) Update(getter BitcoinGetter, initState State, latestHeight uint) error {
	state := initState
	curHeight := state.height
	for i := curHeight + 1; i <= latestHeight; i++ {
		ordTransfer, err := getter.GetOrdTransfers(i)
		if err != nil {
			return err
		}
		state = processOrdTransfer(state.Copy(), ordTransfer)
		hash, err := getter.GetBlockHash(i)
		if err != nil {
			return err
		}
		state.height = i
		state.hash = hash
		queue.Offer(state)
	}
	return nil
}

// Check if the reorganization happened.
// If so, return the height where the reorganization happened, else, return 0.
func (queue *StateQueue) CheckForReorg(getter BitcoinGetter) (uint, error) {
	for i := 0; i <= len(queue.states)-1; i++ {
		state := queue.states[i]
		height := state.height
		hash := state.hash
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
