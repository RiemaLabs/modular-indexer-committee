package apis

import (
	"bytes"
	"encoding/base64"
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

func ParseStateDiff(Keys [][]byte, PreValues, PostValues [][]byte) *verkle.StateDiff {
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

func VerifyCurrentBalanceOfPkscript(preRootC *verkle.Point, tick, pkscript string, resp *Brc20VerifiableCurrentBalanceOfPkscriptResponse) (bool, error) {
	availKey := stateless.GetTickPkscriptHash(tick, ord.Pkscript(pkscript), stateless.AvailableBalancePkscript)
	overallKey := stateless.GetTickPkscriptHash(tick, ord.Pkscript(pkscript), stateless.OverallBalancePkscript)

	availValue, err := ParseBalance(resp.Result.AvailableBalance)
	if err != nil {
		return false, err
	}
	overallValue, err := ParseBalance(resp.Result.OverallBalance)
	if err != nil {
		return false, err
	}

	vProof, err := ParseProof(*resp.Proof)
	if err != nil {
		return false, err
	}

	stateDiff := ParseStateDiff([][]byte{availKey, overallKey}, [][]byte{availValue, overallValue}, [][]byte{{}, {}})

	proof, err := verkle.DeserializeProof(vProof, *stateDiff)
	if err != nil {
		return false, err
	}

	preRoot, err := verkle.PreStateTreeFromProof(proof, preRootC)
	if err != nil {
		return false, err
	}

	err = verkle.VerifyVerkleProofWithPreState(proof, preRoot)
	if err != nil {
		return false, err
	}

	return true, nil
}

func VerifyCurrentBalanceOfWallet(rootC *verkle.Point, tick, wallet string, resp *Brc20VerifiableCurrentBalanceOfWalletResponse) (bool, error) {
	pkscript := resp.Result.Pkscript
	respWallet := Brc20VerifiableCurrentBalanceOfPkscriptResponse{
		Error: resp.Error,
		Result: &Brc20VerifiableCurrentBalanceOfPkscriptResult{
			AvailableBalance: resp.Result.AvailableBalance,
			OverallBalance:   resp.Result.OverallBalance,
		},
		Proof: resp.Proof,
	}
	return VerifyCurrentBalanceOfPkscript(rootC, tick, pkscript, &respWallet)
}

func GenerateCorrectPostRoot(rootC *verkle.Point, blockHeight uint, resp *Brc20VerifiableLatestStateProofResponse) (verkle.VerkleNode, error) {
	preProofBytes, _ := base64.StdEncoding.DecodeString(*resp.Proof)
	preVerkleProof := &verkle.VerkleProof{}
	preVerkleProof.UnmarshalJSON(preProofBytes)

	preProof, err := verkle.DeserializeProof(preVerkleProof, nil)
	if nil != err {
		return nil, err
	}

	parentTree, err := verkle.PreStateTreeFromProof(preProof, rootC)
	if nil != err {
		return nil, err
	}

	if err := verkle.VerifyVerkleProofWithPreState(preProof, parentTree); err != nil {
		return nil, err
	}

	preState := &stateless.LightHeader{
		Root:   parentTree,
		Height: blockHeight - 1,
		Hash:   "",
	}

	var ordTransfers []getter.OrdTransfer
	for _, tran := range resp.Result.OrdTransfers {
		contentBytes, _ := base64.StdEncoding.DecodeString(tran.Content)
		ordTransfers = append(ordTransfers, getter.OrdTransfer{
			ID:            tran.ID,
			InscriptionID: tran.InscriptionID,
			OldSatpoint:   tran.NewSatpoint,
			NewSatpoint:   tran.NewSatpoint,
			NewPkscript:   tran.NewPkscript,
			NewWallet:     tran.NewWallet,
			SentAsFee:     tran.SentAsFee,
			Content:       contentBytes,
			ContentType:   tran.ContentType,
		})
	}

	stateless.Exec(preState, ordTransfers, blockHeight)
	return preState.Root, nil
}
