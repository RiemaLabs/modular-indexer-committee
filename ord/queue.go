package ord

import (
	"log"

	"github.com/RiemaLabs/indexer-committee/ord/getter"
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
		Diff: 	newDiff,
	}
}

func (queue *Queue) StartHeight() uint {
	return queue.History[0].Height
}

func (queue *Queue) LastestHeight() uint {
	return queue.Header.Height
}

func (queue *Queue) GerDiffAtHeight(height uint) DiffState {
	curHeight := queue.Header.Height
	// if height >= curHeight || height < curHeight-5{
		// return nil
	// }
	curLen := len(queue.History)
	return queue.History[curLen-(int(curHeight-height))]
}

// Offer the latest state and pop the oldest state.
func (queue *Queue) Offer() {
	// Offer is not given parameter to protect from wrong writing
    newDiffState := DiffState{
        Height: queue.Header.Height,
        Hash:   queue.Header.Hash,
        Diff:   queue.Header.Temp,
    }

    copy(queue.History[:], queue.History[1:])
    queue.History[len(queue.History)-1] = newDiffState
}

func (queue *Queue) Println() {
	log.Println("====", queue.Header.Height, "====", queue.Header.Hash, "====")
	for _, node := range queue.History {
		log.Print(node.Height, "*")
	}
}

func (queue *Queue) Update(getter getter.OrdGetter, latestHeight uint) error {
	curHeight := queue.Header.Height
	for i := curHeight + 1; i <= latestHeight; i++ {
		ordTransfer, err := getter.GetOrdTransfers(i)
		if err != nil {
			return err
		}
		Exec(&queue.Header, ordTransfer)
		queue.Offer()
		queue.Header.Paging(getter, true, NodeResolveFn)
	}
	queue.Println()
	return nil
}


func (queue *Queue) Recovery(getter getter.OrdGetter, recoveryTillHeight uint) error {
	curHeight := queue.Header.Height
	startHeight := queue.StartHeight()

	for i := curHeight-1; i >= recoveryTillHeight-1; i-- {
		// Recover header from i
		pastState := queue.GerDiffAtHeight(i)
		queue.Header.Height = i
		queue.Header.Hash = pastState.Hash

		for _, elem := range pastState.Diff.Elements {
			queue.Header.KV[elem.Key] = elem.OldValue[:]
			queue.Header.Root.Insert(elem.Key[:], elem.OldValue[:], NodeResolveFn)
		}
	}

	for j := recoveryTillHeight-1; j < curHeight; j++ {
		index := j - startHeight
		ordTransfer, err := getter.GetOrdTransfers(j+1)
		if err != nil {
			return err
		}
		Exec(&queue.Header, ordTransfer)
		var hash string
		hash, err = getter.GetBlockHash(j)
		if err != nil {
			return err
		}
		queue.History[index] = DiffState{
            Height: j,
            Hash:   hash,
            Diff:   queue.Header.Temp,
        }
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
	var stateList [BitcoinConfirmations-1]DiffState
	for i := startHeight; i <= startHeight+BitcoinConfirmations-2; i++ {
		ordTransfer, err := getter.GetOrdTransfers(i)
		if err != nil {
			return nil, err
		}
		Exec(header, ordTransfer)
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
            Diff:   header.Temp,
        }
		header.Paging(getter, true, NodeResolveFn)
	}
	queue := Queue{
		Header: *header,
		History: stateList,
	}
	return &queue, nil
}