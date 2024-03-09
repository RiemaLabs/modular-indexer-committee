package main

import (
	// "bytes"
	// "encoding/gob"
	"encoding/json"
	"strconv"
	// "errors"
	"time"
	"math/big"
	"log"

	// base58 "github.com/btcsuite/btcd/btcutil/base58"
	// bech32 "github.com/btcsuite/btcd/btcutil/bech32"
	verkle "github.com/ethereum/go-verkle"
	uint256 "github.com/holiman/uint256"

	// "gorm.io/datatypes"
	// "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var BRC20StartBlock uint = 779832
var MAXLEN int = 10
var upperLimit = getLimit()
var CURHEIGHT = 779832

type brc20BlockHash struct {
	BlockHeight      int            `gorm:"type:int;not null"`
	BlockHash		 string         `gorm:"type:text;not null"`

}

type VerkleHiostory struct {
	elements []verkle.VerkleNode
	maxLen   int
}

// NewVerkleHiostory: create a VerkleHiostory
func NewVerkleHiostory(maxLen int) *VerkleHiostory {
	return &VerkleHiostory{
		elements: make([]verkle.VerkleNode, 0, maxLen),
		maxLen:   maxLen,
	}
}

// Push to the VerkleHistory
func (q *VerkleHiostory) Push(element verkle.VerkleNode) {
	if len(q.elements) >= q.maxLen {
		q.elements = q.elements[1:]
	}
	q.elements = append(q.elements, element)
}

// Pop the element from the VerkleHostory, only used when the reorganization happens
func (q *VerkleHiostory) PopBack() (verkle.VerkleNode, bool) {
	if len(q.elements) == 0 {
		return nil, false // When the VerkleHistory is emtpy, return false
	}
	index := len(q.elements) - 1
	element := q.elements[index]
	q.elements = q.elements[:index]
	return element, true
}

// Three APIs
func (q *VerkleHiostory)brc20_verifiable_get_current_balance_of_wallet(tick string, Addr string, newPkscript string) {
	// （获取indexer最新的余额数据和证明）（给一些tick+钱包/tick+pkscript
	index := len(q.elements) - 1
	curRoot := q.elements[index]

	// Get Proof needed
	// TODO: how to generate proof， check 对吗
	// proof, _, _, _, _ := verkle.MakeVerkleMultiProof(preState, postState, keys, nodeResolveFn)
	preState := q.elements[index-1]
	curEvents := getBRC20EventAtHeight(db, uint(CURHEIGHT), defaultTick) // 这里没有defaultTick啊？？
	keys, postState := execute(preState, curEvents)
	proof, _, _, _, _ := verkle.MakeVerkleMultiProof(preState, postState, keys, nodeResolveFn)



	availableKey, overallKey := getHash("available-balance", tick, newPkscript), getHash("overall-balance", tick, newPkscript)
	resAvail, resOver := uint256.NewInt(0), uint256.NewInt(0)
	valueAvail, _ := curRoot.Get(availableKey, nodeResolveFn)
	valueOver, _ := curRoot.Get(overallKey, nodeResolveFn)

	if len(valueAvail) != 0 {
		resAvail.SetBytes(valueAvail)
	}
	if len(valueAvail) != 0 {
		resOver.SetBytes(valueOver)
	}
	if len(valueAvail) != 0 && len(valueAvail) != 0 {
		return proof, resAvail, resOver
	}

	availableKey2, overallKey2 = getHash("available-balance", tick, newAddr), getHash("overall-balance", tick, newAddr)
	resAvail2, resOver2 := uint256.NewInt(0), uint256.NewInt(0)
	valueAvail2, _ := curRoot.Get(availableKey2, nodeResolveFn)
	valueOver2, _ := curRoot.Get(overallKey2, nodeResolveFn)

	if len(valueAvail2) != 0 {
		resAvail2.SetBytes(valueAvail2)
	}
	if len(valueAvail2) != 0 {
		resOver2.SetBytes(valueOver2)
	}

	return proof, resAvail2, resOver2

}

func (q *VerkleHiostory)brc20_verifiable_block_height() {
	// (获取indexer最新同步的BRC-20区块高度）（也是一个key_value）
	// blockHeightKey := getHash(strconv.Itoa(CURHEIGHT), "", "")
	// TODO check 对吗
	return CURHEIGHT
}

func (q *VerkleHiostory)brc20_verifiable_get_current_statediff() {
	// 获取indexer最新上传的checkpoint和上一个checkpoint的状态变化)(要保存10个verkle tree，10个历史状态)
	// TODO 怎么做, 需要先了解proof怎么弄
	
}




func processBlockHash(db *gorm.DB, stateRoot verkle.VerkleNode, blockHeight uint) verkle.VerkleNode {
	// Read from the brc20BlockHash and then insert into the verkle tree
	var blockhashes []brc20BlockHash
	db.Where("block_height = ?", blockHeight).Unscoped().Find(&blockhashes)
	blockHeightKey := getHash(strconv.Itoa(blockhashes[0].BlockHeight), "", "")
	// TODO: 不确定这样子做可不可以，就是我把hash 再hash了一次
	hashblockhash := getHash(blockhashes[0].BlockHash, "", "")
	stateRoot.Insert(blockHeightKey, hashblockhash, nodeResolveFn) 
	return stateRoot
}

func processBalanceAtHeight(db *gorm.DB, stateRoot verkle.VerkleNode, blockHeight uint) verkle.VerkleNode {
	var balances []BRC20HistoricBalances
	db.Where("block_height = ?", blockHeight).Unscoped().Find(&balances)

	for _, balance := range balances {
		newPkscript, newAddr, tick, amountString := balance.Pkscript, balance.Wallet, balance.Tick, balance.AvailableBalance
		// TODO: amount should be availableBalance or totalBalance, or either is OK
		keyTick, keyRS, keyLPM, keyD := getTickHash(tick)
		tickExists, _ := stateRoot.Get(keyTick, nodeResolveFn)
		if len(tickExists) == 0 {
			continue // not deployed
		}
		remainingSupplyBytes, _ := stateRoot.Get(keyRS, nodeResolveFn)
		limitPerMintBytes, _ := stateRoot.Get(keyLPM, nodeResolveFn)
		decimalsBytes, _ := stateRoot.Get(keyD, nodeResolveFn)
		remainingSupply := convertByteToInt(remainingSupplyBytes)
		limitPerMint := convertByteToInt(limitPerMintBytes)
		decimals := convertByteToInt(decimalsBytes)
		if !isPositiveNumberWithDot(amountString, false) {
			continue // invalid amount
		}
		amount, err := getNumberExtendedTo18Decimals(amountString, decimals, false)
		if err != nil || amount == nil {
			continue // invalid amount
		}
		if amount.Gt(upperLimit) || amount.IsZero() {
			continue // invalid amount
		}
		if remainingSupply.IsZero() {
			continue // mint ended
		}
		if limitPerMint != nil && amount.Gt(limitPerMint) {
			continue // mint too much
		}
		if amount.Gt(remainingSupply) {
			amount.Set(remainingSupply) // mint remaining token
		}

		stateRoot = mintInscribe(stateRoot, blockHeight, "", newPkscript, newAddr, tick, amount)
	}

	return stateRoot
}

func processTickAtHeight(db *gorm.DB, stateRoot verkle.VerkleNode, blockHeight uint) verkle.VerkleNode {
	/*
	processTickAtHeight collect all Tick at Height, it should be processed prior to the processBalanceAtHeight
	*/
	var deployedTicks []BRC20Tickers
	db.Where("block_height=?", blockHeight).Unscoped().Find(&deployedTicks)

	for _, deployedTick := range deployedTicks {
		tick, maxSupplyValue, decValue, lim  := deployedTick.Tick, deployedTick.MaxSupply, deployedTick.Decimals, deployedTick.LimitPerMint

		// Get decimals through conversion From decValue
		decimals := uint256.NewInt(18)
		if !isPositiveNumber(decValue, false) {
			continue // invalid decimals
		} else {
			
			decimalsInt, err := strconv.Atoi(decValue)
			if err != nil {
				continue
			}
			decimals, _ = uint256.FromBig(big.NewInt(int64(decimalsInt)))
		}

		// Get maxSupply through conversion From maxSupplyValue
		var maxSupply *uint256.Int
		maxSupply, err := getNumberExtendedTo18Decimals(maxSupplyValue, decimals, false)
		if err != nil || maxSupply == nil {
			continue // invalid max supply
		}
		if maxSupply.Gt(upperLimit) || maxSupply.IsZero() {
			continue // invalid max supply
		}

		limitPerMint := maxSupply
		if !isPositiveNumberWithDot(lim, false) {
			continue // invalid limit per mint
		} else {
			limitPerMint, err := getNumberExtendedTo18Decimals(lim, decimals, false)
			if err != nil || limitPerMint == nil {
				continue // invalid limit per mint
			}
			if limitPerMint.Gt(upperLimit) || limitPerMint.IsZero() {
				continue // invalid limit per mint
			}
		}
		
		stateRoot = deployInscribe(stateRoot, blockHeight, "", "", "", tick, maxSupply, decimals, limitPerMint)
	}

	return stateRoot
}

// TODO: complete the following function
func processEventAtHeight(db *gorm.DB, stateRoot verkle.VerkleNode, blockHeight uint) verkle.VerkleNode {
	// Process the transfer event on blockchain
	var events []BRC20Events
	db.Where("block_height = ?", blockHeight).Unscoped().Find(&events)
	for _, e := range events {
		var event map[string]interface{}
		json.Unmarshal(e.Event, &event)
		
		// first check if it is a transfer event
		eventtype := e.EventType
		if eventtype == 0 || eventtype == 1 {
			continue
		}

		tick := event["tick"].(string)
		amountString := event["amount"].(string)

		if eventtype == 2 {

		} else if eventtype == 3 {

		}
		
		
		inscrId, newPkscript, newAddr, txId := e.InscriptionID, 


		keyTick, _, _, keyD := getTickHash(tick)
		tickExists, _ := stateRoot.Get(keyTick, nodeResolveFn)
		decimalBytes, _ := stateRoot.Get(keyD, nodeResolveFn)
		if len(tickExists) == 0 {
			continue // not deployed
		}
		deicmals := convertByteToInt(decimalBytes)
		if !isPositiveNumberWithDot(amountString, false) {
			continue // invalid amount
		}
		amount, err := getNumberExtendedTo18Decimals(amountString, deicmals, false)
		if err != nil || amount == nil {
			continue // invalid amount
		}
		if amount.Gt(upperLimit) || amount.IsZero() {
			continue // invalid amount
		}
		// check if available balance is enough
		// TODO: oldSatpoint是什么,如何处理剩下来的，不是每一个函数都需要全部的variable,需要重写一下varible
		if oldSatpoint == "" {
			availableBalance := getValueOrZero(stateRoot, getHash("available-balance", tick, newPkscript))

			if availableBalance.Lt(amount) {
				continue // not enough available balance
			} else {
				stateRoot = transferInscribe(stateRoot, blockHeight, inscrId, newPkscript, newAddr, tick, amount, availableBalance)
			}
		} else {
			if isUsedOrInvalid(stateRoot, inscrId) {
				continue // already used or invalid
			}
			if sentAsFee {
				stateRoot = transferTransferSpendToFee(stateRoot, blockHeight, inscrId, tick, amount, txId)
			} else {
				stateRoot = transferTransferNormal(stateRoot, blockHeight, inscrId, newPkscript, newAddr, tick, amount, txId)
			}
		}
	}
	return stateRoot
}

func (verkles *VerkleHiostory) updateAtHeight(db *gorm.DB, blockHeight uint){
	index := len(verkles.elements) - 1
	var tmpVerkle = verkles.elements[index]
	tmpVerkle = processBlockHash(db, tmpVerkle, height)
	tmpVerkle = processTickAtHeight(db, tmpVerkle, height)
	tmpVerkle = processBalanceAtHeight(db, tmpVerkle, height)
	tmpVerkle = processEventAtHeight(db, tmpVerkle, height)
	verkles.Push(tmpVerkle)
	// TODO: Upload tmpVerkle to the S3
	return tmpVerkle
}

// Initialize the Verkle Tree since BRC20StartBlock
func (verkles *VerkleHiostory) initCommittee(db *gorm.DB) {
	log.Println("Initize the Verkle Tree at Height ?!", BRC20StartBlock)
	
	// one less than the current height, TODO: delete later after the OPI is running on the server
	var latestHeight uint = 831944

	for height := BRC20StartBlock; height <= latestHeight; height += 1 {
		log.Println("Integrating Blocks at Height ?!", height)

		verkles.updateAtHeight(db, height)
		log.Println("Successfully Integrating Blocks at Height ?!", height)
		CURHEIGHT = height
	}
}

func check_for_update(db *gorm.DB, stateRoot verkle.VerkleNode) bool{
	var maxBlockHeight int
	err := db.Model(&BRC20HistoricBalances{}).Select("max(block_height)").Scan(&maxBlockHeight).Error
	if err != nil {
    	log.Printf("Error finding max block_height: %v", err)
	}
	if maxBlockHeight > CURHEIGHT {
		log.Print("New Height Detected")
		return true
	}
	return false
}

func check_for_reorg(db *gorm.DB, stateRoot verkle.VerkleNode) int{
	// check if reorgnization happened, return the number of verkle trees need to be updated
	var needToBeUpdated int = 0
	for i := CURHEIGHT; i>=CURHEIGHT-10; i-- {
		var blockhashes []brc20BlockHash
		db.Where("block_height = ?", CURHEIGHT).Unscoped().Find(&blockhashes)
		blockHeightKey := getHash(strconv.Itoa(blockhashes[0].BlockHeight), "", "")
		hashblockhash := getHash(blockhashes[0].BlockHash, "", "")
		prevhashblockhash, _ := stateRoot.Get(blockHeightKey, nodeResolveFn)
		if prevhashblockhash == hashblockhash {
			return needToBeUpdated
		} else {
			needToBeUpdated ++
			if i == CURHEIGHT-10 {
				log.Println("Critical Error")
			}
		}
	}
	return needToBeUpdated
}

func (verkles *VerkleHiostory) updateCommittee(db *gorm.DB) {
	for {
		time.Sleep(10 * time.Second) // Update every 10 seconds
		// check_for_reorg first
		index := len(verkles.elements) - 1
		needToBeUpdated := check_for_reorg(db, verkles.elements[index])
		if needToBeUpdated > 10 {
			log.Println("Critical Error")
			break
		} else if needToBeUpdated < 10 && needToBeUpdated > 0 {
			// log.Println("? reorgnization detected", needToBeUpdated)
			for i := 0; i < needToBeUpdated; i++ {
				verkles.PopBack()
				CURHEIGHT --
			}
			// log.Println("Verkle History Cleared")
			for i := CURHEIGHT; i < CURHEIGHT+needToBeUpdated; i++ {
				verkles.updateAtHeight(db, height)
				CURHEIGHT ++
			}
			log.Println("Verkle Reorgnization Updated")
		} else {
			// log.Println("No reorgnization detected")
		}
	
		index := len(verkles.elements) - 1
		var tmpVerkle = verkles.elements[index]
		if check_for_update(db, verkles.elements[index]) {
			verkles.updateAtHeight(db, height)
			CURHEIGHT ++
			log.Println("New Height Detected and Updated")
		}
	}
}

func main() {
	db := ConnectDatabase()
	verkles := NewVerkleHiostory(MAXLEN) // A total len of 10 latest verkle trees are stored
	verkles.initCommittee(db)
	// verkleHistory = initCommittee(db)
}

