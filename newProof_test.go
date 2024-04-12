package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/RiemaLabs/modular-indexer-committee/ord"
	"github.com/RiemaLabs/modular-indexer-committee/ord/stateless"
	"github.com/ethereum/go-verkle"
)

func Test_NewProof(t *testing.T) {
	var latestHeight uint = stateless.BRC20StartHeight + ord.BitcoinConfirmations
	ordGetterTest, arguments := loadMain()
	queue, err := CatchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, latestHeight)
	if err != nil {
		log.Fatalf(fmt.Sprintf("error happened: %v", err))
	}
	ordGetterTest.LatestBlockHeight = latestHeight
	go ServiceStage(ordGetterTest, &arguments, queue, 10*time.Millisecond)
	for {
		if ordGetterTest.LatestBlockHeight == queue.LatestHeight() {
			proofExists, proofsEqual := VerifyProof(queue)
			if !proofExists {
				log.Printf("Either of two proofs does not exist!\n")
			} else if proofsEqual {
				log.Printf("Block: %d is verified!\n", ordGetterTest.LatestBlockHeight)
			} else {
				log.Printf("Block: %d cannot pass verification!\n", ordGetterTest.LatestBlockHeight)
			}
			ordGetterTest.LatestBlockHeight++
		}
		if ordGetterTest.LatestBlockHeight >= 800000 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func VerifyProof(queue *stateless.Queue) (bool, bool) {
	// first bool indicates if both Proof exist, second bool indicates if two Proof are equal

	// generate VerifyProof Proof
	if queue.LastStateProof == nil {
		// log.Println("queue.LastStateProof == nil")
		return false, false
	}
	vProof, _, err := verkle.SerializeProof(queue.LastStateProof)
	if err != nil {
		log.Println("[VerifyProof]: verkle.SerializeProof(queue.LastStateProof) failed")
		return false, false
	}
	vProofBytes, err := vProof.MarshalJSON()
	if err != nil {
		return false, false
	}
	finalproof := base64.StdEncoding.EncodeToString(vProofBytes[:])
	// log.Println("VerifyProof finalproof:", finalproof)

	// generate RollingbackProof Proof
	rollingBackProof, exists := RollingbackProof(queue)
	if !exists {
		return false, false
	}
	if finalproof != rollingBackProof {
		return true, false
	}
	return true, true
}

func RollingbackProof(queue *stateless.Queue) (string, bool) {
	// copy most code from apis.GetLatestStateProof
	// and then return the finalproof
	lastIndex := len(queue.History) - 1
	postState := queue.Header.Root
	preState, keys := stateless.Rollingback(queue.Header, &queue.History[lastIndex])

	if len(keys) == 0 {
		log.Println("[RollingbackProof]: len(keys) == 0")
		return "", false
	}

	proofOfKeys, _, _, _, err := verkle.MakeVerkleMultiProof(preState, postState, keys, stateless.NodeResolveFn)
	if err != nil {
		log.Printf("Failed to generate proof due to %v", err)
		return "", false
	}

	vProof, _, err := verkle.SerializeProof(proofOfKeys)
	if err != nil {
		log.Printf("Failed to serialize proof due to %v", err)
		return "", false
	}

	vProofBytes, err := vProof.MarshalJSON()
	if err != nil {
		log.Printf("Failed to marshal the proof to JSON due to %v", err)
		return "", false
	}

	finalproof := base64.StdEncoding.EncodeToString(vProofBytes[:])
	// log.Println("RollingbackProof finalproof:", finalproof)
	return finalproof, true
}

// // 添加到 IPAProof 结构体中
// func (ip *ipa.IPAProof) String() string {
// 	var clStrings, crStrings []string
// 	for _, cl := range ip.CL {
// 		clStrings = append(clStrings, hex.EncodeToString(cl[:]))
// 	}
// 	for _, cr := range ip.CR {
// 		crStrings = append(crStrings, hex.EncodeToString(cr[:]))
// 	}
// 	finalEvaluationString := hex.EncodeToString(ip.FinalEvaluation[:])

// 	return fmt.Sprintf("CL: %v, CR: %v, FinalEvaluation: %s", clStrings, crStrings, finalEvaluationString)
// }

// // 修改 verkleProofMarshaller 结构体的 MarshalJSON 方法
// func (vp *verkle.VerkleProof) MarshalJSON() ([]byte, error) {
// 	aux := &verkleProofMarshaller{
// 		OtherStems:            make([]string, len(vp.OtherStems)),
// 		DepthExtensionPresent: HexToPrefixedString(vp.DepthExtensionPresent),
// 		CommitmentsByPath:     make([]string, len(vp.CommitmentsByPath)),
// 		D:                     HexToPrefixedString(vp.D[:]),
// 		IPAProof:              vp.IPAProof,
// 	}

// 	for i, s := range vp.OtherStems {
// 		aux.OtherStems[i] = HexToPrefixedString(s[:])
// 	}
// 	for i, c := range vp.CommitmentsByPath {
// 		aux.CommitmentsByPath[i] = HexToPrefixedString(c[:])
// 	}

// 	// 在序列化 IPAProof 字段时调用新的打印方法
// 	aux.IPAProof = vp.IPAProof.String()

// 	return json.Marshal(aux)
// }
// type ipaproofMarshaller struct {
//     CL              [8]string `json:"cl"`
//     CR              [8]string `json:"cr"`
//     FinalEvaluation string                  `json:"finalEvaluation"`
// }

// func (ipp *ipa.IPAProof) UnmarshalJSON(data []byte) error {
//     var ipm ipa.ipaproofMarshaller
//     if err := json.Unmarshal(data, &ipm); err != nil {
//         return err
//     }

//     for i, cl := range ipm.CL {
//         bytes, err := hex.DecodeString(cl)
//         if err != nil {
//             return err
//         }
//         copy(ipp.CL[i][:], bytes)
//     }

//     for i, cr := range ipm.CR {
//         bytes, err := hex.DecodeString(cr)
//         if err != nil {
//             return err
//         }
//         copy(ipp.CR[i][:], bytes)
//     }

//     finalEvaluationBytes, err := hex.DecodeString(ipm.FinalEvaluation)
//     if err != nil {
//         return err
//     }
//     copy(ipp.FinalEvaluation[:], finalEvaluationBytes)

//     return nil
// }
