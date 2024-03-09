package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"log"

	base58 "github.com/btcsuite/btcd/btcutil/base58"
	bech32 "github.com/btcsuite/btcd/btcutil/bech32"
	verkle "github.com/ethereum/go-verkle"
	uint256 "github.com/holiman/uint256"

	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type BRC20HistoricBalances struct {
	gorm.Model
	ID               uint   `gorm:"primary_key;auto_increment"`
	Pkscript         string `gorm:"type:text;not null"`
	Wallet           string `gorm:"type:text;not null"`
	Tick             string `gorm:"type:varchar(4);not null"`
	OverallBalance   string `gorm:"type:numeric(40);not null"`
	AvailableBalance string `gorm:"type:numeric(40);not null"`
	BlockHeight      int    `gorm:"type:int;not null"`
	EventID          int64  `gorm:"type:bigint;not null"`
}

type BRC20HistoricBalancesNoID struct {
	Pkscript         string
	Wallet           string
	Tick             string
	OverallBalance   string
	AvailableBalance string
	BlockHeight      int
	EventID          int64
}

type BRC20Events struct {
	gorm.Model
	ID            uint           `gorm:"primary_key;auto_increment"`
	EventType     int            `gorm:"type:int;not null"`
	BlockHeight   int            `gorm:"type:int;not null"`
	InscriptionID string         `gorm:"type:text;not null"`
	Event         datatypes.JSON `gorm:"type:jsonb;not null"`
}

var nodeResolveFn verkle.NodeResolverFn = nil

func ConnectDatabase() *gorm.DB {
	dsn := "host=127.0.0.1 user=postgres password=170501 dbname=postgres port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	return db
}

func convertToNoIDStruct(balancesWithID []BRC20HistoricBalances) []BRC20HistoricBalancesNoID {
	var balancesNoID []BRC20HistoricBalancesNoID
	for _, balance := range balancesWithID {
		balanceNoID := BRC20HistoricBalancesNoID{
			Pkscript:         balance.Pkscript,
			Wallet:           balance.Wallet,
			Tick:             balance.Tick,
			OverallBalance:   balance.OverallBalance,
			AvailableBalance: balance.AvailableBalance,
			BlockHeight:      balance.BlockHeight,
			EventID:          balance.EventID,
		}
		balancesNoID = append(balancesNoID, balanceNoID)
	}
	return balancesNoID
}

func serialize(balance BRC20HistoricBalancesNoID) []byte {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(balance)
	if err != nil {
		log.Fatalf("Failed to serialize BRC20HistoricBalances: %v", err)
	}
	return buf.Bytes()
}

func deserialize(data []byte) BRC20HistoricBalancesNoID {
	var balance BRC20HistoricBalancesNoID
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	err := decoder.Decode(&balance)
	if err != nil {
		log.Fatalf("Failed to deserialize BRC20HistoricBalances: %v", err)
	}
	return balance
}

func getBRC20BalancesAtHeight(db *gorm.DB, blockHeight uint, tick string) []BRC20HistoricBalancesNoID {
	var balances []BRC20HistoricBalances
	db.Where("block_height = ?", blockHeight).
		Where("tick = ?", tick).Unscoped().Find(&balances)
	return convertToNoIDStruct(balances)
}

func getBRC20EventAtHeight(db *gorm.DB, blockHeight uint, tick string) []BRC20Events {
	var events []BRC20Events
	db.Where("block_height = ?", blockHeight).Where("event->>'tick' = ?", tick).Unscoped().Find(&events)
	return events
}

// padTo32Bytes takes a byte slice and, if it's shorter than 32 bytes, pads it with zeros until it reaches 32 bytes in length.
func padTo32Bytes(data []byte) ([]byte, error) {
	if len(data) > 32 {
		return nil, errors.New("data length greater than 32 bytes")
	}
	if len(data) == 32 {
		return data, nil // Already 32 bytes, no padding needed.
	}
	// Create a slice of 32 bytes and copy the data into the beginning of it.
	paddedData := make([]byte, 32)
	copy(paddedData, data)
	// The rest will automatically be zeros, as make initializes slice elements to the zero value of the element type.
	return paddedData, nil
}

func decodeBitcoinAddress(address string) ([]byte, error) {
	hrp, data, errBech32 := bech32.Decode(address)
	if errBech32 == nil && hrp == "bc" {
		// 32 bytes or 20 bytes
		decoded, err := bech32.ConvertBits(data[1:], 5, 8, false)
		if err != nil {
			return nil, err
		}
		decoded, _ = padTo32Bytes(decoded)
		return decoded, nil
	}

	decoded := base58.Decode(address)
	if len(decoded) > 0 {
		decoded, _ = padTo32Bytes(decoded)
		return decoded, nil
	}

	return nil, errors.New("invalid or unsupported bitcoin address format")
}

func convertIntToByte(i *uint256.Int) []byte {
	var dest [32]byte
	i.WriteToArray32(&dest)
	return dest[:]
}

func initStateFromBalances(balances []BRC20HistoricBalancesNoID) verkle.VerkleNode {
	root := verkle.New()
	for _, balance := range balances {
		key, _ := decodeBitcoinAddress(balance.Wallet)
		num, err := uint256.FromDecimal(balance.AvailableBalance)
		if err != nil {
			continue
		}
		data := convertIntToByte(num)
		root.Insert(key, data, nodeResolveFn)
	}
	return root
}

func execute(preStateRoot verkle.VerkleNode, curEvents []BRC20Events) ([][]byte, verkle.VerkleNode) {
	postStateRoot := preStateRoot.Copy()
	var keys [][]byte
	for _, e := range curEvents {
		eType := e.EventType
		if eType == 1 {
			var event map[string]interface{}
			json.Unmarshal(e.Event, &event)
			key, _ := decodeBitcoinAddress(event["minted_wallet"].(string))
			v, err := preStateRoot.Get(key, nodeResolveFn)
			if err != nil {
				// Insert
				// newValue := BRC20HistoricBalancesNoID{
				// 	Pkscript:         event["minted_pkScript"].(string),
				// 	Wallet:           event["minted_wallet"].(string),
				// 	Tick:             event["tick"].(string),
				// 	OverallBalance:   event["amount"].(string),
				// 	AvailableBalance: event["amount"].(string),
				// 	BlockHeight:      e.BlockHeight,
				// 	EventID:          int64(e.ID),
				// }
				// v = serialize(newValue)
				// postStateRoot.Insert(key, v, nodeResolveFn)

				// Insert
				keys = append(keys, key)
				num, err := uint256.FromDecimal(event["amount"].(string))
				if err != nil {
					continue
				}
				data := convertIntToByte(num)
				postStateRoot.Insert(key, data, nodeResolveFn)
			} else {
				// Update
				// keys = append(keys, key)
				// curValue := deserialize(v)
				// overallBal, _ := strconv.Atoi(curValue.OverallBalance)
				// availableBal, _ := strconv.Atoi(curValue.AvailableBalance)
				// amount, _ := strconv.Atoi(event["amount"].(string))
				// newOverallBal := strconv.Itoa(overallBal + amount)
				// newAvailableBal := strconv.Itoa(availableBal + amount)
				// newValue := BRC20HistoricBalancesNoID{
				// 	Pkscript:         event["minted_pkScript"].(string),
				// 	Wallet:           event["minted_wallet"].(string),
				// 	Tick:             event["tick"].(string),
				// 	OverallBalance:   newOverallBal,
				// 	AvailableBalance: newAvailableBal,
				// 	BlockHeight:      e.BlockHeight,
				// 	EventID:          int64(e.ID),
				// }
				// v = serialize(newValue)
				// postStateRoot.Insert(key, v, nodeResolveFn)

				// Update
				keys = append(keys, key)
				curNum := new(uint256.Int).SetBytes(v)
				addOn, err := uint256.FromDecimal(event["amount"].(string))
				if err != nil {
					continue
				}
				num := curNum.Add(curNum, addOn)
				data := convertIntToByte(num)
				postStateRoot.Insert(key, data, nodeResolveFn)
			}
		}
	}
	return keys, postStateRoot
}

func findFirstOccurrence(list [][]byte, target []byte) int {
	for i, v := range list {
		if bytes.Equal(v, target) {
			return i
		}
	}
	return -1
}

func executeStateless(keys, preValues, postValues [][]byte, curEvents []BRC20Events) bool {
	for _, e := range curEvents {
		eType := e.EventType
		if eType == 1 {
			// var event map[string]interface{}
			// json.Unmarshal(e.Event, &event)
			// key, _ := decodeBitcoinAddress(event["minted_wallet"].(string))
			// idx := findFirstOccurrence(proof.Keys, key)
			// if idx == -1 {
			// 	return false
			// }
			// preBin := proof.PreValues[idx]
			// preValue := deserialize(preBin)
			// overallBal, _ := strconv.Atoi(preValue.OverallBalance)
			// availableBal, _ := strconv.Atoi(preValue.AvailableBalance)
			// amount, _ := strconv.Atoi(event["amount"].(string))
			// newOverallBal := strconv.Itoa(overallBal + amount)
			// newAvailableBal := strconv.Itoa(availableBal + amount)
			// postValue := BRC20HistoricBalancesNoID{
			// 	Pkscript:         event["minted_pkScript"].(string),
			// 	Wallet:           event["minted_wallet"].(string),
			// 	Tick:             event["tick"].(string),
			// 	OverallBalance:   newOverallBal,
			// 	AvailableBalance: newAvailableBal,
			// 	BlockHeight:      e.BlockHeight,
			// 	EventID:          int64(e.ID),
			// }
			// postBin := serialize(postValue)
			// if bytes.Equal(postBin, proof.PostValues[idx]) {
			// 	return false
			// }

			var event map[string]interface{}
			json.Unmarshal(e.Event, &event)
			key, _ := decodeBitcoinAddress(event["minted_wallet"].(string))
			idx := findFirstOccurrence(keys, key)
			if idx == -1 {
				return false
			}

			preBin := preValues[idx]
			curNum := new(uint256.Int).SetBytes(preBin)

			addOn, err := uint256.FromDecimal(event["amount"].(string))
			if err != nil {
				return false
			}
			num := curNum.Add(curNum, addOn)
			data := convertIntToByte(num)
			if !bytes.Equal(data, postValues[idx]) {
				return false
			}
		}
	}
	return true
}

func demo() {
	log.Println("Start!")
	db := ConnectDatabase()

	defaultHeight := uint(800000)
	defaultTick := "sats"

	curBalances := getBRC20BalancesAtHeight(db, defaultHeight, defaultTick)

	curEvents := getBRC20EventAtHeight(db, defaultHeight+1, defaultTick)

	// Indexer holds all states.
	preState := initStateFromBalances(curBalances)

	// The preCheckpoint of preState, stored at DA.
	preCheckpoint := preState.Commit()

	// Indexer executes the operations.
	keys, postState := execute(preState, curEvents)

	// The checkpoint of postState, stored at DA.
	postCheckpoint := postState.Commit()

	// Now, the postState is asked to be proved because postCheckpointA and postCheckpointB are different.
	proof, _, _, _, _ := verkle.MakeVerkleMultiProof(preState, postState, keys, nodeResolveFn)

	// Light client downloads the checkpoint from DA and the proof from indexer.
	preStatePartial, _ := verkle.PreStateTreeFromProof(proof, preCheckpoint)

	// Light clients computes the partial postState from the partial preState.
	// Then verifies: partial postState->partial preState is consistent with the stateDiff in the proofOfStateTrans.
	ok := executeStateless(proof.Keys, proof.PreValues, proof.PostValues, curEvents)
	if !ok {
		log.Print("Failed to verify stateDiff, stateDiff is corrupted!")
		return
	}

	// Light clients verifies the next checkpoint.
	_, stateDiff, _ := verkle.SerializeProof(proof)

	postStatePartial, _ := verkle.PostStateTreeFromStateDiff(preStatePartial, stateDiff)

	postCheckpointPartial := postStatePartial.Commit()

	if !postCheckpoint.Equal(postCheckpointPartial) {
		log.Print("Failed to verify checkpoint, checkpoint is corrupted!")
		return
	}

	log.Println("Succeed to verify the checkpoint!")

	// Light client gets some state variables from indexers and verifies them by the checkpoint.
	start := 100
	end := 116
	postKeys := proof.Keys[start:end]
	postValues := proof.PostValues[start:end]

	// Indexer sends the proof of key-value pairs.
	proofOfKeys, _, _, _, _ := verkle.MakeVerkleMultiProof(postState, nil, postKeys, nodeResolveFn)

	// Light client computes the partial state.
	postStatePartial, _ = verkle.PreStateTreeFromProof(proofOfKeys, postCheckpoint)
	if err := verkle.VerifyVerkleProofWithPreState(proofOfKeys, postStatePartial); err != nil {
		log.Print("Failed to verify verkle proof, verkle proof is corrupted!")
		return
	}

	// Get the value from the proof.
	pe, _, _, _ := verkle.GetCommitmentsForMultiproof(postStatePartial, postKeys, nodeResolveFn)
	postValuesRet := pe.Vals

	if len(postValues) != len(postValuesRet) {
		log.Print("Failed to get the value!")
		return
	}

	for i, a := range postValues {
		if !bytes.Equal(a, postValuesRet[i]) {
			log.Print("Failed to get the value!")
			return
		}
	}

	log.Println("Succeed to get the values!")

	// Serialize & Deserialize
	vProof, stateDiff, _ := verkle.SerializeProof(proof)
	vProofBytes, _ := vProof.MarshalJSON()
	var stateDiffBytes [][]byte
	for _, s := range stateDiff {
		sBin, _ := s.MarshalJSON()
		stateDiffBytes = append(stateDiffBytes, sBin)
	}
	var vProofD verkle.VerkleProof
	vProofD.UnmarshalJSON(vProofBytes)
	var stateDiffD []verkle.StemStateDiff
	for _, s := range stateDiffBytes {
		var ss verkle.StemStateDiff
		ss.UnmarshalJSON(s)
		stateDiffD = append(stateDiffD, ss)
	}
	proofD, _ := verkle.DeserializeProof(&vProofD, stateDiffD)
	println(proofD)
}
