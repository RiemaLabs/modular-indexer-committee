package main

import (
	"log"
	"time"

	verkle "github.com/ethereum/go-verkle"
	uint256 "github.com/holiman/uint256"
	"golang.org/x/crypto/sha3"
	"gorm.io/gorm"
)

type BRC20StateDiff struct {
	Key   string
	Value []byte
}

type BlockStatsData struct {
	Height       uint
	GlobalKeyNum int
	UpdateKeyNum int
	UpdateTime   time.Duration
}

type InitType struct {
	VerkleRoot verkle.VerkleNode
	VerkleSize int
	TimeCost   time.Duration
}

// Get hash value by keccak256(“available_balance” + “keccak256("tick_name")” + "keccak256("wallet_address")")
func getHash(prefix string, tick string, pkScript string) []byte {
	prefixBytes := []byte(prefix)
	tickData := []byte(tick)
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(tickData)
	tickHash := hasher.Sum(nil)
	pkScriptData := []byte(pkScript)
	hasher = sha3.NewLegacyKeccak256()
	hasher.Write(pkScriptData)
	pkScriptHash := hasher.Sum(nil)
	hasher = sha3.NewLegacyKeccak256()
	hasher.Write(append(append(prefixBytes, tickHash...), pkScriptHash...))
	return hasher.Sum(nil)
}

// Get different states and diffrent states K-V number
func getBRC20StateDiff(db *gorm.DB, blockHeight uint) ([]BRC20StateDiff, int) {
	var diffBalances []BRC20HistoricBalances
	db.Where("block_height = ?", blockHeight).Unscoped().Find(&diffBalances)
	diffBalancesNoID := convertToNoIDStruct(diffBalances)
	var diffState []BRC20StateDiff
	for _, diff := range diffBalancesNoID {
		availableBalance, err := uint256.FromDecimal(diff.AvailableBalance)
		if err != nil {
			log.Println("[Get State Diff] Error: fail to run uint256.FromDecimal(balance.AvailableBalance)")
			continue
		}
		diffState = append(diffState, BRC20StateDiff{
			Key:   string(getHash("available-balance", diff.Tick, diff.Wallet)),
			Value: availableBalance.Bytes(),
		})
	}
	// deployedTicksAtHeight := getDeployedTicksAtHeight(db, blockHeight)
	// diffState = append(diffState, deployedTicksAtHeight...)
	return diffState, len(diffState)
}

// Return initial Verkle Tree, initial key number, construct time
func initBRC20State(db *gorm.DB, startHeight uint, blockHeight uint) (verkle.VerkleNode, int, time.Duration) {
	log.Println("Start initializeing block ", blockHeight)
	stateDiffMap := make(map[string][]byte)
	for height := startHeight; height <= blockHeight; height += 1 {
		stateDiff, _ := getBRC20StateDiff(db, height)
		for _, diff := range stateDiff {
			stateDiffMap[diff.Key] = diff.Value
		}
		if height%100 == 0 {
			log.Println("Block ", height, " finished")
		}
	}
	// deployedTicks := getDeployedTicks(db, blockHeight)
	// for _, diff := range deployedTicks {
	// 	stateDiffMap[diff.Key] = diff.Value
	// }
	// Construct initial Verkle Tree
	startTime := time.Now()
	root := verkle.New()
	for k, v := range stateDiffMap {
		root.Insert([]byte(k), v, nodeResolveFn)
	}
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	log.Println("Initializeing block ", blockHeight, " finished")
	return root, len(stateDiffMap), duration
}

// Update Verkle Tree and get the global state key number, update time
func updateBRC20State(root verkle.VerkleNode, stateDiff []BRC20StateDiff, prevSize int) (int, time.Duration) {
	size := prevSize
	startTime := time.Now()
	// Update on previous state root directly
	for _, diff := range stateDiff {
		key := []byte(diff.Key)
		value, _ := root.Get(key, nodeResolveFn)
		if len(value) == 0 {
			size += 1
		}
		root.Insert(key, diff.Value, nodeResolveFn)
	}
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	return size, duration
}

// Entry function of Demo A, return an array contains the global key number, update key number and update time of each block
func indexBRC20Database(initBlockHeight uint, blockNum uint, startBlockHeight uint) []BlockStatsData {
	db := ConnectDatabase()
	stateRoot, initKeyNum, initTime := initBRC20State(db, startBlockHeight, initBlockHeight)

	var res []BlockStatsData
	totalGlobalKeyNum, totalUpdateKeyNum, totalUpdateTime := 0, 0, time.Duration(0)
	globalKeyNum := initKeyNum
	var updateTime time.Duration
	for i := initBlockHeight + 1; i <= initBlockHeight+blockNum; i += 1 {
		stateDiff, updateKeyNum := getBRC20StateDiff(db, i)
		globalKeyNum, updateTime = updateBRC20State(stateRoot, stateDiff, globalKeyNum)
		totalUpdateKeyNum += updateKeyNum
		totalGlobalKeyNum += globalKeyNum
		totalUpdateTime += updateTime
		res = append(res, BlockStatsData{
			Height:       i,
			GlobalKeyNum: globalKeyNum,
			UpdateKeyNum: updateKeyNum,
			UpdateTime:   updateTime,
		})
	}

	avgGlobalKeyNum := float64(totalGlobalKeyNum) / float64(blockNum)
	avgUpdateKeyNum := float64(totalUpdateKeyNum) / float64(blockNum)
	avgUpdateTime := time.Duration(float64(totalUpdateTime) / float64(blockNum))
	log.Println("Initial Global Key Number: ", initKeyNum)
	log.Println("Average Global Key Number: ", avgGlobalKeyNum)
	log.Println("Average Update Key Number: ", avgUpdateKeyNum)
	log.Println("Initialize Verkle Tree Time: ", initTime)
	log.Println("Average Update Verkle Tree Time: ", avgUpdateTime)
	return res
}
