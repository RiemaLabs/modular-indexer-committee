package stateless

import (
	"log"

	"github.com/RiemaLabs/indexer-committee/ord"
	"github.com/RiemaLabs/indexer-committee/ord/getter"
	verkle "github.com/ethereum/go-verkle"
)

func (state DiffState) Copy() DiffState {
	newElements := make([]TripleElement, len(state.Diff.Elements))

	for i, elem := range state.Diff.Elements {
		newElements[i] = TripleElement{
			Key:      elem.Key,
			OldValue: elem.OldValue,
			NewValue: elem.NewValue,
		}
	}

	newDiff := DiffList{Elements: newElements}
	return DiffState{
		Height: state.Height,
		Hash:   state.Hash,
		Diff:   newDiff,
	}
}

func (queue *Queue) StartHeight() uint {
	return queue.History[0].Height
}

func (queue *Queue) LastestHeight() uint {
	return queue.Header.Height
}

// Offer the latest state and pop the oldest state.
func (queue *Queue) Offer() {
	// Offer is not given parameter to protect from wrong writing
	newDiffState := DiffState{
		Height: queue.Header.Height,
		Hash:   queue.Header.Hash,
		Diff:   queue.Header.Diff,
	}

	copy(queue.History[:], queue.History[1:])
	queue.History[len(queue.History)-1] = newDiffState
}

func (queue *Queue) Println() {
	log.Println("====", queue.Header.Height, "====", queue.Header.Hash, "====")
	for _, node := range queue.History {
		log.Print(node.Height, "*", node.Hash)
	}
}

func (queue *Queue) Update(getter getter.OrdGetter, latestHeight uint) error {
	curHeight := queue.Header.Height
	for i := curHeight + 1; i <= latestHeight; i++ {
		ordTransfer, err := getter.GetOrdTransfers(i)
		if err != nil {
			return err
		}
		Exec(&queue.Header, ordTransfer, i)
		queue.Offer()
		queue.Header.OrdTrans = ordTransfer
		queue.Header.Paging(getter, true, NodeResolveFn)
	}
	return nil
}

func Rollingback(root verkle.VerkleNode, stateDiff DiffState) (verkle.VerkleNode, [][]byte, []TripleElement) {
	rollback := root.Copy()
	var keys [][]byte

	for _, elem := range stateDiff.Diff.Elements {
		keys = append(keys, elem.Key[:])
		if elem.OldValueExists {
			rollback.Insert(elem.Key[:], elem.OldValue[:], NodeResolveFn)
		} else {
			rollback.Delete(elem.Key[:], NodeResolveFn)
		}
	}

	return rollback, keys, stateDiff.Diff.Elements
}

func (queue *Queue) Recovery(getter getter.OrdGetter, recoveryTillHeight uint) error {
	curHeight := queue.Header.Height
	startHeight := queue.StartHeight()

	for i := curHeight - 1; i >= recoveryTillHeight-1; i-- {
		// Recover header from i
		index2 := i - startHeight
		pastState := queue.History[index2]
		queue.Header.Height = i
		queue.Header.Hash = pastState.Hash

		for _, elem := range pastState.Diff.Elements {
			if elem.OldValueExists {
				queue.Header.KV[elem.Key] = elem.OldValue
				queue.Header.Root.Insert(elem.Key[:], elem.OldValue[:], NodeResolveFn)
			} else {
				queue.Header.Root.Delete(elem.Key[:], NodeResolveFn)
				delete(queue.Header.KV, elem.Key)
			}
		}
	}

	log.Print(curHeight, startHeight, recoveryTillHeight)

	for j := recoveryTillHeight - 1; j < curHeight; j++ {
		index := j - startHeight
		ordTransfer, err := getter.GetOrdTransfers(j + 1)
		if err != nil {
			return err
		}
		Exec(&queue.Header, ordTransfer, j+1)
		var hash string
		hash, err = getter.GetBlockHash(j)
		if err != nil {
			return err
		}
		queue.History[index] = DiffState{
			Height: j,
			Hash:   hash,
			Diff:   queue.Header.Diff,
		}
		queue.Header.OrdTrans = ordTransfer
		queue.Header.Paging(getter, true, NodeResolveFn)
	}

	return nil
}

func (queue *Queue) CheckForReorg(getter getter.OrdGetter) (uint, error) {
	// return the height that needs to start reorg
	for i := 0; i <= len(queue.History)-1; i++ {
		state := queue.History[i]
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
	hash := queue.Header.Hash
	height := queue.Header.Height
	newHash, err := getter.GetBlockHash(height)
	if err != nil {
		return 0, err
	}
	if hash == newHash {
		return 0, nil
	} else {
		return height, nil
	}
}

func NewQueues(getter getter.OrdGetter, header *Header, queryHash bool, startHeight uint) (*Queue, error) {
	var stateList [ord.BitcoinConfirmations - 1]DiffState
	for i := startHeight; i <= startHeight+ord.BitcoinConfirmations-2; i++ {
		ordTransfer, err := getter.GetOrdTransfers(i)
		if err != nil {
			return nil, err
		}
		Exec(header, ordTransfer, i)
		var hash string
		if queryHash {
			hash, err = getter.GetBlockHash(i)
			if err != nil {
				return nil, err
			}
		}
		stateList[i-startHeight] = DiffState{
			Height: i,
			Hash:   hash,
			Diff:   header.Diff,
		}
		header.Paging(getter, true, NodeResolveFn)
	}
	queue := Queue{
		Header:  *header,
		History: stateList,
	}
	return &queue, nil
}
