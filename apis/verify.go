package apis

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"unsafe"

	"github.com/RiemaLabs/modular-indexer-committee/ord"
	"github.com/RiemaLabs/modular-indexer-committee/ord/getter"
	"github.com/RiemaLabs/modular-indexer-committee/ord/stateless"
	"github.com/ethereum/go-verkle"
	"github.com/holiman/uint256"
)

func ParseBalance(balance string) ([]byte, error) {
	value, err := uint256.FromDecimal(balance)
	if err != nil {
		return []byte{}, err
	}
	var dest [stateless.ValueSize]byte
	value.WriteToArray32(&dest)
	return dest[:], err
}

func ParseProof(proof string) (*verkle.VerkleProof, error) {
	vProofBytes, err := base64.StdEncoding.DecodeString(proof)
	if err != nil {
		return nil, err
	}
	var vProof verkle.VerkleProof
	err = vProof.UnmarshalJSON(vProofBytes)
	if err != nil {
		return nil, err
	}
	return &vProof, nil
}

func ParseCommitment(commitment string) (*verkle.Point, error) {
	bytes, err := base64.StdEncoding.DecodeString(commitment)
	if err != nil {
		return nil, err
	}
	var p verkle.Point
	err = p.SetBytes(bytes)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func ParseStateDiff(Keys, PreValues, PostValues [][]byte) *verkle.StateDiff {
	var stemdiff *verkle.StemStateDiff
	var statediff verkle.StateDiff
	for i, key := range Keys {
		stem := verkle.KeyToStem(key)
		if stemdiff == nil || !bytes.Equal(stemdiff.Stem[:], stem) {
			statediff = append(statediff, verkle.StemStateDiff{})
			stemdiff = &statediff[len(statediff)-1]
			copy(stemdiff.Stem[:], stem)
		}
		stemdiff.SuffixDiffs = append(stemdiff.SuffixDiffs, verkle.SuffixStateDiff{Suffix: key[verkle.StemSize]})
		newsd := &stemdiff.SuffixDiffs[len(stemdiff.SuffixDiffs)-1]

		var valueLen = len(PreValues[i])
		switch valueLen {
		case 0:
			// null value
		case 32:
			newsd.CurrentValue = (*[32]byte)(PreValues[i])
		default:
			var aligned [32]byte
			copy(aligned[:valueLen], PreValues[i])
			newsd.CurrentValue = (*[32]byte)(unsafe.Pointer(&aligned[0]))
		}

		valueLen = len(PostValues[i])
		switch valueLen {
		case 0:
			// null value
		case 32:
			newsd.NewValue = (*[32]byte)(PostValues[i])
		default:
			// TODO remove usage of unsafe
			var aligned [32]byte
			copy(aligned[:valueLen], PostValues[i])
			newsd.NewValue = (*[32]byte)(unsafe.Pointer(&aligned[0]))
		}
	}
	return &statediff
}
func VerifyCurrentBalanceOfWallet(rootC *verkle.Point, tick, wallet string, resp *Brc20VerifiableCurrentBalanceOfWalletResponse) (bool, error) {
	if resp.Error != nil {
		return false, fmt.Errorf("failed to obtain the proof from committee indexer, error: %s", *resp.Error)
	}
	availKey := stateless.GetTickWalletHash(tick, ord.Wallet(wallet), stateless.AvailableBalanceWallet)
	overallKey := stateless.GetTickWalletHash(tick, ord.Wallet(wallet), stateless.OverallBalanceWallet)

	keys := [][]byte{availKey, overallKey}

	availValue, err := ParseBalance(resp.Result.AvailableBalance)
	if err != nil {
		return false, err
	}
	overallValue, err := ParseBalance(resp.Result.OverallBalance)
	if err != nil {
		return false, err
	}

	values := [][]byte{availValue, overallValue}

	vProof, err := ParseProof(*resp.Proof)
	if err != nil {
		return false, err
	}

	stateDiff := ParseStateDiff(keys, values, [][]byte{{}, {}})

	preProof, err := verkle.DeserializeProof(vProof, *stateDiff)
	if err != nil {
		return false, err
	}

	preRoot, err := verkle.PreStateTreeFromProof(preProof, rootC)
	if err != nil {
		return false, err
	}

	err = verkle.VerifyVerkleProofWithPreState(preProof, preRoot)
	if err != nil {
		return false, err
	}

	return true, nil
}

func GeneratePostRoot(rootC *verkle.Point, blockHeight uint, resp *Brc20VerifiableLatestStateProofResponse) (verkle.VerkleNode, error) {
	if resp.Error != nil {
		return nil, fmt.Errorf("failed to generate the post root at block height %d from committee indexer, error: %s", blockHeight, *resp.Error)
	}

	var preRoot verkle.VerkleNode
	if resp.Proof != nil {
		preProofBytes, _ := base64.StdEncoding.DecodeString(*resp.Proof)
		preVerkleProof := &verkle.VerkleProof{}
		_ = preVerkleProof.UnmarshalJSON(preProofBytes)

		stateDiff := make([]verkle.StemStateDiff, 0)
		for _, s := range resp.Result.StateDiff {
			bytes, err := base64.StdEncoding.DecodeString(s)
			if err != nil {
				return nil, fmt.Errorf("failed to generate the post root at block height %d from committee indexer, error: %s", blockHeight, *resp.Error)
			}
			var sd verkle.StemStateDiff
			err = sd.UnmarshalJSON(bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to generate the post root at block height %d from committee indexer, error: %s", blockHeight, *resp.Error)
			}
			stateDiff = append(stateDiff, sd)
		}

		preProof, err := verkle.DeserializeProof(preVerkleProof, stateDiff)
		if err != nil {
			return nil, err
		}
		preRoot, err = verkle.PreStateTreeFromProof(preProof, rootC)
		if err != nil {
			return nil, err
		}
		if err := verkle.VerifyVerkleProofWithPreState(preProof, preRoot); err != nil {
			return nil, err
		}
	} else {
		preRoot = verkle.NewStatelessInternal(0, rootC)
	}

	preRoot.Commit()

	preHeader := &stateless.LightHeader{
		Root:   preRoot,
		Height: blockHeight - 1,
		Hash:   "",
	}

	if resp.Result == nil {
		return preHeader.Root, nil
	}

	var ordTransfers []getter.BRC20Event
	for _, item := range resp.Result.OrdTransfers {
		data, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("type assertion to map[string]interface{} failed")
		}
		event, err := convertMapToBRC20Event(data)
		if err != nil {
			return nil, fmt.Errorf("type transfer failed")
		}
		ordTransfers = append(ordTransfers, event)
	}

	stateless.Exec(preHeader, ordTransfers, blockHeight)
	return preHeader.Root, nil
}
