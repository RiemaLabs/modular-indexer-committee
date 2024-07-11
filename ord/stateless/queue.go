package stateless

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/RiemaLabs/modular-indexer-committee/internal/tree"
	"github.com/RiemaLabs/modular-indexer-committee/ord"
	"github.com/RiemaLabs/modular-indexer-committee/ord/getter"
	goipa "github.com/crate-crypto/go-ipa"
	"github.com/crate-crypto/go-ipa/common"
	"github.com/crate-crypto/go-ipa/ipa"
	verkle "github.com/RiemaLabs/go-verkle"
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
			VerkleCommit: queue.Header.Root.VerkleTree.Commit().Bytes(),
		}
		copy(queue.History[:], queue.History[1:])
		queue.History[len(queue.History)-1] = newDiffState

		proof, err := generateProofFromUpdate(queue.Header, &newDiffState)
		if err != nil {
			return err
		}
		if proof != nil {
			queue.LastStateProof = proof
		}

		queue.Header.OrdTrans = ordTransfer
		_ = queue.Header.Paging(getter, true)
	}
	return nil
}

func (queue *Queue) Recovery(getter getter.OrdGetter, reorgHeight uint) error {
	queue.Lock()
	defer queue.Unlock()
	// turn off current LevelDB, and remove tmpStore, create from left cache
	queue.Header.Root.KvStore.Close()
	os.RemoveAll(VerkleDataPath)
	// Copy the old LevelDB to the new LevelDB
	
	myHeader := Header{
		Root:           tree.NewVerkleTreeWithLRU(LRUsize, FlushDepth, VerkleDataPath),
		Height:         BRC20StartHeight - 1,
		Access:         AccessList{},
		IntermediateKV: KeyValueMap{},
	}
	queue.Header = &myHeader
	directories, err := os.ReadDir(CachePath)
	if err != nil {
		return nil
	}
	// Variables to keep track of the file with the maximum state.height
	var maxHeight int
	var maxDir string

	// Iterate through all files
	for _, dir := range directories {
		if dir.IsDir() && filepath.Ext(dir.Name()) == FileSuffix {
			heightString := strings.TrimSuffix(dir.Name(), FileSuffix)
			height, err := strconv.Atoi(heightString)
			if err == nil && height > maxHeight {
				maxHeight = height
				maxDir = dir.Name()
			}
		}
	}
	if maxDir != "" && uint(maxHeight) <= reorgHeight {
		storedState, err := Deserialize(uint(maxHeight))
		if err != nil {
			return nil
		}
		queue.Header = storedState
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
			VerkleCommit: header.Root.VerkleTree.Commit().Bytes(),
		}
		if i == startHeight+ord.BitcoinConfirmations-1 {
			proof, _ = generateProofFromUpdate(header, &stateList[i-startHeight])
		}
		_ = header.Paging(getter, true)
	}
	// The call of Commit is necessary to refresh the root commit.
	header.Root.VerkleTree.Commit()
	queue := Queue{
		Header:         header,
		History:        stateList,
		LastStateProof: proof,
	}
	return &queue, nil
}

func generateProofFromUpdate(header *Header, stateDiff *DiffState) (*verkle.Proof, error) {
	if len(stateDiff.Access.Elements) == 0 {
		return nil, nil
	}
	var keys [][]byte
	kvMap := make(KeyValueMap)
	for _, elem := range stateDiff.Access.Elements {
		keys = append(keys, elem.Key[:])
		kvMap[elem.Key] = elem.NewValue
	}

	preroot := header.Root.VerkleTree
	pe, es, poas, err := verkle.GetCommitmentsForMultiproof(preroot, keys, header.Root.KvStore.Get)
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
		src := pe.ByPath[path]
		dst := *src
		cis[i] = &dst
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
