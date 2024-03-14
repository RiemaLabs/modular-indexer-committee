package main

import (
	"encoding/base64"
	"log"
	"strconv"
	"sync"

	verkle "github.com/ethereum/go-verkle"
)

type State struct {
	root   verkle.VerkleNode
	height uint
	hash   string
}

// Historical state neither uploads Checkpoint nor records hash.
func (state *State) HasHash() bool {
	return state.hash != ""
}

// Maintain a queue of states to prepare for the re-org.
type StateQueues struct {
	states [BitcoinConfirmations]State
	sync.RWMutex
}

// Build the queue from the start height.
func NewQueues(getter BitcoinGetter, initStateRoot verkle.VerkleNode, queryHash bool, startHeight uint) (*StateQueues, error) {
	var states [BitcoinConfirmations]State
	stateRoot := initStateRoot
	for i := startHeight; i <= startHeight+BitcoinConfirmations-1; i++ {
		ordTransfer, err := getter.GetOrdTransfers(i)
		if err != nil {
			return nil, err
		}
		stateRoot = processOrdTransfer(stateRoot, ordTransfer, i).Copy()
		var hash string
		if queryHash {
			hash, err = getter.GetBlockHash(i)
			if err != nil {
				return nil, err
			}
		} else {
			hash = ""
		}
		state := State{
			root:   stateRoot,
			height: i,
			hash:   hash,
		}
		states[i] = state
	}
	queue := StateQueues{
		states: states,
	}
	return &queue, nil
}

func (queue *StateQueues) StartHeight() uint {
	return queue.states[0].height
}

func (queue *StateQueues) LastestHeight() uint {
	return queue.StartHeight() + uint(len(queue.states))
}

func (queue *StateQueues) LastestStateRoot() verkle.VerkleNode {
	return queue.states[len(queue.states)-1].root
}

func (queue *StateQueues) StateRoot(blockHeight uint) verkle.VerkleNode {
	return queue.states[blockHeight-queue.StartHeight()].root
}

// Offer the latest state and pop the oldest state.
func (queue *StateQueues) Offer(element State) {
	for i := 0; i <= len(queue.states)-2; i++ {
		queue.states[i] = queue.states[i+1]
	}
	queue.states[len(queue.states)-1] = element
}

func (queue *StateQueues) Println() {
	log.Println("====", len(queue.states), "====", queue.StartHeight(), "====")
	for _, node := range queue.states {
		log.Print(node.height, "*")
	}
}

func (queue *StateQueues) Update(getter BitcoinGetter, initStateRoot verkle.VerkleNode, startHeight uint, latestHeight uint) error {
	curHeight := startHeight
	stateRoot := initStateRoot
	for i := curHeight; i <= latestHeight; i++ {
		ordTransfer, err := getter.GetOrdTransfers(i)
		if err != nil {
			return err
		}
		stateRoot = processOrdTransfer(stateRoot, ordTransfer, i).Copy()
		hash, err := getter.GetBlockHash(i)
		if err != nil {
			return err
		}
		state := State{
			root:   stateRoot,
			height: i,
			hash:   hash,
		}
		queue.Offer(state)
	}
	return nil
}

// Check if the reorganization happened.
// If so, return the height where the reorganization happened, else, return 0.
func (queue *StateQueues) CheckForReorg(getter BitcoinGetter) (uint, error) {
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

// TODO: Refine the update performance.
// We have not caught up with the latest progress yet, we are more than 10 blocks behind the latest block.
func FastCatchup(getter BitcoinGetter, initStateRoot verkle.VerkleNode, curHeight uint, catchupHeight uint) (verkle.VerkleNode, error) {
	log.Printf("Fast catchup to the lateset block height! From %d to %d \n", curHeight, catchupHeight)

	stateRoot := initStateRoot

	for i := curHeight; i <= catchupHeight; i++ {
		if i%100 == 0 {
			log.Printf("Blocks: %d / %d \n", i, catchupHeight)
		}
		ordTransfer, err := getter.GetOrdTransfers(i)
		if err != nil {
			return nil, err
		}
		stateRoot = processOrdTransfer(stateRoot, ordTransfer, i)
	}
	return stateRoot, nil
}
