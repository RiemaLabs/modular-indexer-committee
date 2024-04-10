package stateless

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"sort"

	"github.com/RiemaLabs/modular-indexer-committee/ord"
	"github.com/RiemaLabs/modular-indexer-committee/ord/getter"
	goipa "github.com/crate-crypto/go-ipa"
	"github.com/crate-crypto/go-ipa/common"
	"github.com/crate-crypto/go-ipa/ipa"
	verkle "github.com/ethereum/go-verkle"
)

func (state DiffState) Copy() DiffState {
	newElements := make([]TripleElement, len(state.Access.Elements))

	for i, elem := range state.Access.Elements {
		newElements[i] = TripleElement{
			Key:      elem.Key,
			OldValue: elem.OldValue,
			NewValue: elem.NewValue,
		}
	}

	newDiff := AccessList{Elements: newElements}
	return DiffState{
		Height:       state.Height,
		Hash:         state.Hash,
		Access:       newDiff,
		VerkleCommit: state.VerkleCommit,
	}
}

func (queue *Queue) StartHeight() uint {
	return queue.History[0].Height
}

func (queue *Queue) LatestHeight() uint {
	return queue.Header.Height
}

func (queue *Queue) Println() {
	log.Println("====", queue.Header.Height, "====", queue.Header.Hash, "====")
	for _, node := range queue.History {
		log.Print(node.Height, "*", node.Hash)
	}
}

func (queue *Queue) Update(getter getter.OrdGetter, latestHeight uint) error {
	queue.Lock()
	defer queue.Unlock()
	curHeight := queue.Header.Height
	for i := curHeight + 1; i <= latestHeight; i++ {
		ordTransfer, err := getter.GetOrdTransfers(i)
		if err != nil {
			return err
		}
		// Write to Diff
		Exec(queue.Header, ordTransfer, i)
		hash, err := getter.GetBlockHash(i - 1)
		if err != nil {
			return err
		}
		newDiffState := DiffState{
			Height:       i - 1,
			Hash:         hash,
			Access:       queue.Header.Access,
			VerkleCommit: queue.Header.Root.Commit().Bytes(),
		}
		copy(queue.History[:], queue.History[1:])
		queue.History[len(queue.History)-1] = newDiffState

		proof, _ := generateProofFromUpdate(queue.Header, &newDiffState)
		queue.LastStateProof = proof

		queue.Header.OrdTrans = ordTransfer
		// header.Height ++
		queue.Header.Paging(getter, true, NodeResolveFn)
	}
	return nil
}

func Rollingback(header *Header, stateDiff *DiffState) (verkle.VerkleNode, [][]byte) {
	var keys [][]byte
	kvMap := make(KeyValueMap)
	for k, v := range header.KV {
		kvMap[k] = v
	}

	for _, elem := range stateDiff.Access.Elements {
		keys = append(keys, elem.Key[:])
		if elem.OldValueExists {
			kvMap[elem.Key] = elem.OldValue
		} else {
			delete(kvMap, elem.Key)
		}
	}

	rollback := verkle.New()
	for k, v := range kvMap {
		rollback.Insert(k[:], v[:], NodeResolveFn)
	}
	// The call of Commit is necessary to refresh the root commit.
	rollback.Commit()

	return rollback, keys
}

func (queue *Queue) Recovery(getter getter.OrdGetter, reorgHeight uint) error {
	queue.Lock()
	defer queue.Unlock()
	curHeight := queue.Header.Height
	startHeight := queue.StartHeight()

	// Rollback to the reorgHeight - 1.
	for i := curHeight - 1; i >= reorgHeight-1; i-- {
		index := i - startHeight
		pastState := queue.History[index]

		// Inner bug in go-verkle, doesn't work.
		// for _, elem := range pastState.Diff.Elements {
		// 	if elem.OldValueExists {
		// 		queue.Header.KV[elem.Key] = elem.OldValue
		// 		queue.Header.Root.Insert(elem.Key[:], elem.OldValue[:], NodeResolveFn)
		// 	} else {
		// 		delete(queue.Header.KV, elem.Key)
		// 		queue.Header.Root.Delete(elem.Key[:], NodeResolveFn)
		// 	}
		// }
		// newRoot := queue.Header.Root
		// newBytes := queue.Header.Root.Commit().Bytes()
		// n := base64.StdEncoding.EncodeToString(newBytes[:])

		for _, elem := range pastState.Access.Elements {
			if elem.OldValueExists {
				queue.Header.KV[elem.Key] = elem.OldValue
			} else {
				delete(queue.Header.KV, elem.Key)
			}
		}
		newRoot := verkle.New()
		for k, v := range queue.Header.KV {
			newRoot.Insert(k[:], v[:], NodeResolveFn)
		}
		newBytes := newRoot.Commit().Bytes()
		n := base64.StdEncoding.EncodeToString(newBytes[:])
		o := base64.StdEncoding.EncodeToString(pastState.VerkleCommit[:])
		if n != o {
			panic(fmt.Sprintf("Recovery the header failed! The commitment is different: %s and %s", n, o))
		}
		newHeader := Header{
			Root:           newRoot,
			KV:             queue.Header.KV,
			Height:         i,
			Hash:           pastState.Hash,
			Access:         AccessList{},
			IntermediateKV: KeyValueMap{},
			OrdTrans:       queue.Header.OrdTrans,
		}
		queue.Header = &newHeader
	}

	// Compute to the curHeight from the reorgHeight.
	for i := reorgHeight; i <= curHeight; i++ {
		index := i - startHeight - 1
		ordTransfer, err := getter.GetOrdTransfers(i)
		if err != nil {
			return err
		}
		Exec(queue.Header, ordTransfer, i)
		var hash string
		hash, err = getter.GetBlockHash(i - 1)
		if err != nil {
			return err
		}
		queue.History[index] = DiffState{
			Height:       i - 1,
			Hash:         hash,
			Access:       queue.Header.Access,
			VerkleCommit: queue.Header.Root.Commit().Bytes(),
		}
		queue.Header.OrdTrans = ordTransfer
		queue.Header.Paging(getter, true, NodeResolveFn)
	}

	return nil
}

func (queue *Queue) CheckForReorg(getter getter.OrdGetter) (uint, error) {
	queue.Lock()
	defer queue.Unlock()
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
	return 0, nil
}

func NewQueues(getter getter.OrdGetter, header *Header, queryHash bool, startHeight uint) (*Queue, error) {
	var stateList [ord.BitcoinConfirmations]DiffState
	var proof *verkle.Proof
	for i := startHeight; i <= startHeight+ord.BitcoinConfirmations-1; i++ {
		ordTransfer, err := getter.GetOrdTransfers(i)
		if err != nil {
			return nil, err
		}
		Exec(header, ordTransfer, i)
		var hash string
		if queryHash {
			hash, err = getter.GetBlockHash(i - 1)
			if err != nil {
				return nil, err
			}
		}
		stateList[i-startHeight] = DiffState{
			Height:       i - 1,
			Hash:         hash,
			Access:       header.Access,
			VerkleCommit: header.Root.Commit().Bytes(),
		}
		if i == startHeight+ord.BitcoinConfirmations-1 {
			proof, _ = generateProofFromUpdate(header, &stateList[i-startHeight])
		}
		header.Paging(getter, true, NodeResolveFn)
	}
	// The call of Commit is necessary to refresh the root commit.
	header.Root.Commit()
	queue := Queue{
		Header:         header,
		History:        stateList,
		LastStateProof: proof,
	}
	return &queue, nil
}

func generateProofFromUpdate(header *Header, stateDiff *DiffState) (*verkle.Proof, error) {
	if len(stateDiff.Access.Elements) == 0 {
		log.Printf("len(stateDiff.Access.Elements) == 0")
	}
	var keys [][]byte
	kvMap := make(KeyValueMap)
	for _, elem := range stateDiff.Access.Elements {
		keys = append(keys, elem.Key[:])
		kvMap[elem.Key] = elem.NewValue
	}

	if len(keys) == 0 {
		log.Printf("no key provided for proof")
		return nil, nil
	}

	preroot := header.Root
	pe, es, poas, err := verkle.GetCommitmentsForMultiproof(preroot, keys, NodeResolveFn)
	if err != nil {
		return nil, fmt.Errorf("error getting pre-state proof data: %w", err)
	}

	postvals := make([][]byte, len(keys))
	// keys were sorted already in the above GetcommitmentsForMultiproof.
	// Set the post values, if they are untouched, leave them `nil`
	for i := range keys {
		val := kvMap[bytesTo32Bytes(keys[i])]
		if !bytes.Equal(pe.Vals[i], val[:]) {
			postvals[i] = val[:]
		}
	}

	// cfg := verkle.GetConfig()
	conf, err := ipa.NewIPASettings()
	if err != nil {
		return nil, fmt.Errorf("creating multiproof: %w", err)
	}
	tr := common.NewTranscript("vt")
	mpArg, err := goipa.CreateMultiProof(tr, conf, pe.Cis, pe.Fis, pe.Zis)
	if err != nil {
		return nil, fmt.Errorf("creating multiproof: %w", err)
	}

	// Copied from verkle-go
	// It's wheel-reinvention time again 🎉: reimplement a basic
	// feature that should be part of the stdlib.
	// "But golang is a high-productivity language!!!" 🤪
	// len()-1, because the root is already present in the
	// parent block, so we don't keep it in the proof.
	paths := make([]string, 0, len(pe.ByPath)-1)
	for path := range pe.ByPath {
		if len(path) > 0 {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	cis := make([]*verkle.Point, len(pe.ByPath)-1)
	for i, path := range paths {
		cis[i] = pe.ByPath[path]
	}

	proof := &verkle.Proof{
		Multipoint: mpArg,
		Cs:         cis,
		ExtStatus:  es,
		PoaStems:   poas,
		Keys:       keys,
		PreValues:  pe.Vals,
		PostValues: postvals,
	}
	return proof, nil
}

func (queue *Queue) VerifyProof() bool {
	if queue.LastStateProof == nil {
		log.Println("queue.LastStateProof == nil")
		return true
	}
	vProof, _, err := verkle.SerializeProof(queue.LastStateProof)
	if err != nil {
		log.Println("[VerifyProof]: verkle.SerializeProof(queue.LastStateProof) failed")
		return false
	}
	vProofBytes, err := vProof.MarshalJSON()
	if err != nil {
		return false
	}
	finalproof := base64.StdEncoding.EncodeToString(vProofBytes[:])
	return finalproof == queue.RollingbackProof()
}
func (queue *Queue) RollingbackProof() string {
	// copy most code from apis.GetLatestStateProof
	// and then return the finalproof
	lastIndex := len(queue.History) - 1
	postState := queue.Header.Root
	preState, keys := Rollingback(queue.Header, &queue.History[lastIndex])

	if len(keys) == 0 {
		log.Println("[RollingbackProof]: len(keys) == 0")
		return ""
	}

	proofOfKeys, _, _, _, err := verkle.MakeVerkleMultiProof(preState, postState, keys, NodeResolveFn)
	if err != nil {
		log.Printf("Failed to generate proof due to %v", err)
		return ""
	}

	vProof, _, err := verkle.SerializeProof(proofOfKeys)
	if err != nil {
		log.Printf("Failed to serialize proof due to %v", err)
		return ""
	}

	vProofBytes, err := vProof.MarshalJSON()
	if err != nil {
		log.Printf("Failed to marshal the proof to JSON due to %v", err)
		return ""
	}

	finalproof := base64.StdEncoding.EncodeToString(vProofBytes[:])
	return finalproof
}
