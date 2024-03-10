package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"log"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-verkle"
	uint256 "github.com/holiman/uint256"
	"gorm.io/gorm"
)

func deployInscribe(stateRoot verkle.VerkleNode, blockHeight uint, inscrId string, newPkscript string, newAddr string, tick string, maxSupply *uint256.Int, decimals *uint256.Int, limitPerMint *uint256.Int) verkle.VerkleNode {
	keyTick, keyRS, keyLPM, keyD := getTickHash(tick)
	stateRoot.Insert(keyTick, convertIntToByte(uint256.NewInt(0)), nodeResolveFn)
	stateRoot.Insert(keyRS, convertIntToByte(maxSupply), nodeResolveFn)
	stateRoot.Insert(keyLPM, convertIntToByte(limitPerMint), nodeResolveFn)
	stateRoot.Insert(keyD, convertIntToByte(decimals), nodeResolveFn)
	return stateRoot
}

func mintInscribe(stateRoot verkle.VerkleNode, blockHeight uint, inscrId string, newPkscript string, newAddr string, tick string, amount *uint256.Int) verkle.VerkleNode {
	newAddrByte, _ := decodeBitcoinAddress(newAddr)
	newAddr = string(newAddrByte)

	// store tick + pkscript
	availableKey, overallKey := getHash("available-balance", tick, newPkscript), getHash("overall-balance", tick, newPkscript)
	prevAvailableBalance, prevOverallBalance := getValueOrZero(stateRoot, availableKey), getValueOrZero(stateRoot, overallKey)
	newAvailableBalance, newOverallBalance := uint256.NewInt(0).Add(prevAvailableBalance, amount), uint256.NewInt(0).Add(prevOverallBalance, amount)
	stateRoot.Insert(availableKey, convertIntToByte(newAvailableBalance), nodeResolveFn)
	stateRoot.Insert(overallKey, convertIntToByte(newOverallBalance), nodeResolveFn)

	// store tick + wallet
	availableKey, overallKey = getHash("available-balance", tick, newAddr), getHash("overall-balance", tick, newAddr)
	stateRoot.Insert(availableKey, convertIntToByte(newAvailableBalance), nodeResolveFn)
	stateRoot.Insert(overallKey, convertIntToByte(newOverallBalance), nodeResolveFn)

	// update tick info
	_, keyRS, _, _ := getTickHash(tick)
	prevRemainingSupply, _ := stateRoot.Get(keyRS, nodeResolveFn)
	newRemainingSupply := uint256.NewInt(0).Sub(convertByteToInt(prevRemainingSupply), amount)
	stateRoot.Insert(keyRS, convertIntToByte(newRemainingSupply), nodeResolveFn)
	return stateRoot
}

func transferInscribe(stateRoot verkle.VerkleNode, blockHeight uint, inscrId string, sourcePkScript string, sourceAddr string, tick string, amount *uint256.Int, availableBalance *uint256.Int) verkle.VerkleNode {
	sourceAddrByte, _ := decodeBitcoinAddress(sourceAddr)
	sourceAddr = string(sourceAddrByte)

	newAvailableBalance := uint256.NewInt(0).Sub(availableBalance, amount)
	availableKey := getHash("available-balance", tick, sourceAddr)
	stateRoot.Insert(availableKey, convertIntToByte(newAvailableBalance), nodeResolveFn)
	availableKey = getHash("available-balance", tick, sourcePkScript)
	stateRoot.Insert(availableKey, convertIntToByte(newAvailableBalance), nodeResolveFn)

	// store transfer-inscribe event
	saveSourceWalletAndPkscript(stateRoot, inscrId, sourceAddr, sourcePkScript)

	// update transfer-inscribe event count
	eventCntKey := getEventHash("transfer-inscribe-count", inscrId)
	newEventCnt := uint256.NewInt(0).Add(getValueOrZero(stateRoot, eventCntKey), uint256.NewInt(1))
	stateRoot.Insert(eventCntKey, convertIntToByte(newEventCnt), nodeResolveFn)

	return stateRoot
}

func isUsedOrInvalid(stateRoot verkle.VerkleNode, inscrId string) bool {
	tIEventKey := getEventHash("transfer-inscribe-count", inscrId)
	transferInscribeCnt := getValueOrZero(stateRoot, tIEventKey)

	tTEventKey := getEventHash("transfer-transfer-count", inscrId)
	transferTransferCnt := getValueOrZero(stateRoot, tTEventKey)

	return !transferInscribeCnt.Eq(uint256.NewInt(1)) || !transferTransferCnt.IsZero()
}

func transferTransferSpendToFee(stateRoot verkle.VerkleNode, blockHeight uint, inscrId string, tick string, amount *uint256.Int, txId uint) verkle.VerkleNode {
	sourceAddr, sourcePkScript := getSourceWalletAndPkscript(stateRoot, inscrId)
	availableKey := getHash("available-balance", tick, sourceAddr)
	lastAvailableBalance := getValueOrZero(stateRoot, availableKey)
	newAvailableBalance := uint256.NewInt(0).Add(lastAvailableBalance, amount)
	stateRoot.Insert(availableKey, convertIntToByte(newAvailableBalance), nodeResolveFn)
	availableKey = getHash("available-balance", tick, sourcePkScript)
	stateRoot.Insert(availableKey, convertIntToByte(newAvailableBalance), nodeResolveFn)

	// update transfer-transfer event count
	eventCntKey := getEventHash("transfer-transfer-count", inscrId)
	newTransferTransferCnt := uint256.NewInt(0).Add(getValueOrZero(stateRoot, eventCntKey), uint256.NewInt(1))
	stateRoot.Insert(eventCntKey, convertIntToByte(newTransferTransferCnt), nodeResolveFn)

	return stateRoot
}

func transferTransferNormal(stateRoot verkle.VerkleNode, blockHeight uint, inscrId string, spentPkScript string, spentAddr string, tick string, amount *uint256.Int, txId uint) verkle.VerkleNode {
	spentAddrByte, _ := decodeBitcoinAddress(spentAddr)
	spentAddr = string(spentAddrByte)

	sourceAddr, sourcePkScript := getSourceWalletAndPkscript(stateRoot, inscrId)
	sourceOverallKey := getHash("overall-balance", tick, sourceAddr)
	newSourceOverallBalance := uint256.NewInt(0).Sub(getValueOrZero(stateRoot, sourceOverallKey), amount)
	stateRoot.Insert(sourceOverallKey, convertIntToByte(newSourceOverallBalance), nodeResolveFn)
	sourceOverallKey = getHash("overall-balance", tick, sourcePkScript)
	stateRoot.Insert(sourceOverallKey, convertIntToByte(newSourceOverallBalance), nodeResolveFn)

	spentAvailableKey, spentOverallKey := getHash("available-balance", tick, spentAddr), getHash("overall-balance", tick, spentAddr)
	newSpentAvailableBalance, newSpentOverallBalance := uint256.NewInt(0).Add(getValueOrZero(stateRoot, spentAvailableKey), amount), uint256.NewInt(0).Add(getValueOrZero(stateRoot, spentOverallKey), amount)
	stateRoot.Insert(spentAvailableKey, convertIntToByte(newSpentAvailableBalance), nodeResolveFn)
	stateRoot.Insert(spentOverallKey, convertIntToByte(newSpentOverallBalance), nodeResolveFn)
	spentAvailableKey, spentOverallKey = getHash("available-balance", tick, spentPkScript), getHash("overall-balance", tick, spentPkScript)
	stateRoot.Insert(spentAvailableKey, convertIntToByte(newSpentAvailableBalance), nodeResolveFn)
	stateRoot.Insert(spentOverallKey, convertIntToByte(newSpentOverallBalance), nodeResolveFn)

	// update transfer-transfer event count
	eventCntKey := getEventHash("transfer-transfer-count", inscrId)
	newTransferTransferCnt := uint256.NewInt(0).Add(getValueOrZero(stateRoot, eventCntKey), uint256.NewInt(1))
	stateRoot.Insert(eventCntKey, convertIntToByte(newTransferTransferCnt), nodeResolveFn)
	return stateRoot
}

// Get all ord transfer records in a block
func getOrdTransfers(db *gorm.DB, height uint) []OrdTransfer {
	var ordTransfer []OrdTransfer
	sql := `
		SELECT ot.id, ot.inscription_id, ot.old_satpoint, ot.new_pkscript, ot.new_wallet, ot.sent_as_fee, oc."content", oc.content_type
		FROM ord_transfers ot
		LEFT JOIN ord_content oc ON ot.inscription_id = oc.inscription_id
		LEFT JOIN ord_number_to_id onti ON ot.inscription_id = onti.inscription_id
		WHERE ot.block_height = ? 
			AND onti.cursed_for_brc20 = false
			AND oc."content" is not null AND oc."content"->>'p' = 'brc-20'
		ORDER BY ot.id asc;
		`
	db.Raw(sql, height).Scan(&ordTransfer)
	return ordTransfer
}

// Input previous verkle tree and all ord records in a block, then get the K-V array that the verkle tree should update
func processOrdTransfer(stateRoot verkle.VerkleNode, ordTransfer []OrdTransfer, blockHeight uint) verkle.VerkleNode {
	upperLimit := getLimit()
	if len(ordTransfer) == 0 {
		return stateRoot
	}
	for _, transfer := range ordTransfer {
		txId, inscrId, oldSatpoint, newPkscript, newAddr, sentAsFee, contentType := transfer.ID, transfer.InscriptionID, transfer.OldSatpoint, transfer.NewPkscript, transfer.NewWallet, transfer.SentAsFee, transfer.ContentType
		var js map[string]string
		json.Unmarshal(transfer.Content, &js)
		if sentAsFee && oldSatpoint == "" {
			continue // inscribed as fee
		}
		if contentType == "" {
			continue // invalid inscription
		}
		decodedBytes, err := hex.DecodeString(contentType)
		if err == nil {
			contentType = string(decodedBytes)
		}
		contentType = strings.Split(contentType, ";")[0]
		if contentType != "application/json" && contentType != "text/plain" {
			continue // invalid inscription
		}
		tick, ok := js["tick"]
		if !ok {
			continue // invalid inscription
		}
		if _, ok := js["op"]; !ok {
			continue // invalid inscription
		}
		tick = strings.ToLower(tick)
		// NOTATION1 different to BRC20
		if len(tick) != 4 {
			continue // invalid tick
		}

		// handle deploy
		if js["op"] == "deploy" && oldSatpoint == "" {
			if tick == "μσ" {
				log.Println("[enter 0]")
			}
			maxSupplyValue, ok := js["max"]
			if !ok {
				continue // invalid inscription
			}
			keyTick, _, _, _ := getTickHash(tick)
			if v, _ := stateRoot.Get(keyTick, nodeResolveFn); len(v) != 0 {
				continue // already deployed
			}
			decimals := uint256.NewInt(18)
			if decValue, ok := js["dec"]; ok {
				if !isPositiveNumber(decValue, false) {
					continue // invalid decimals
				} else {
					decimalsInt, err := strconv.Atoi(decValue)
					if err != nil {
						continue
					}
					decimals, _ = uint256.FromBig(big.NewInt(int64(decimalsInt)))
				}
			}
			if decimals.Gt(uint256.NewInt(18)) {
				continue // invalid decimals
			}
			var maxSupply *uint256.Int
			if !isPositiveNumberWithDot(maxSupplyValue, false) {
				continue
			} else {
				maxSupply, err = getNumberExtendedTo18Decimals(maxSupplyValue, decimals, false)
				if err != nil || maxSupply == nil {
					continue // invalid max supply
				}
				if maxSupply.Gt(upperLimit) || maxSupply.IsZero() {
					continue // invalid max supply
				}
			}
			limitPerMint := maxSupply
			if lim, ok := js["lim"]; ok {
				if !ok {
					continue
				}
				if !isPositiveNumberWithDot(lim, false) {
					continue // invalid limit per mint
				} else {
					limitPerMint, err = getNumberExtendedTo18Decimals(lim, decimals, false)
					if err != nil || limitPerMint == nil {
						continue // invalid limit per mint
					}
					if limitPerMint.Gt(upperLimit) || limitPerMint.IsZero() {
						continue // invalid limit per mint
					}
				}
			}
			stateRoot = deployInscribe(stateRoot, blockHeight, inscrId, newPkscript, newAddr, tick, maxSupply, decimals, limitPerMint)
		}

		// handle mint
		if js["op"] == "mint" && oldSatpoint == "" {
			amountString, ok := js["amt"]
			if !ok {
				continue // invalid inscription
			}
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
			stateRoot = mintInscribe(stateRoot, blockHeight, inscrId, newPkscript, newAddr, tick, amount)
		}

		// handle transfer
		if js["op"] == "transfer" {
			amountString, ok := js["amt"]
			if !ok {
				continue // invalid inscription
			}
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
	}
	return stateRoot
}

func compareServerToOPI(db *gorm.DB, endHeight uint) bool {
	stateRoot := verkle.New()
	initHeight := uint(791113)

	for height := initHeight; height <= endHeight; height += 1 {
		log.Println("[Enter height]: ", height)
		ordTransfer := getOrdTransfers(db, height)
		stateRoot = processOrdTransfer(stateRoot, ordTransfer, height)

		opiDeployedTicks := getDeployedTicksAtHeight(db, height)
		opiStateDiff := getStateDiff(db, height)

		for k, v := range opiStateDiff {
			res, _ := stateRoot.Get([]byte(k), nodeResolveFn)
			if len(res) == 0 {
				log.Println("[No such key at height] ", height)
				log.Println("[No such key]: ", debug[k])
				return false
			}
			if !bytes.Equal(res, v) {
				log.Println("[Inconsistent at height] ", height)
				log.Println("[Inconsistent at key]: ", debug[k])
				log.Println("[value from tree]: ", convertByteToInt(res))
				log.Println("[value from opi]: ", convertByteToInt(v))
				return false
			}
		}

		for k, v := range opiDeployedTicks {
			res, _ := stateRoot.Get([]byte(k), nodeResolveFn)
			if len(res) == 0 {
				log.Println("[Tick: No such key at height] ", height)
				log.Println("[Tick: No such key]: ", debug[k])
				return false
			}
			if !bytes.Equal(res, v) {
				log.Println("[Tick: Inconsistent at height] ", height)
				log.Println("[Tick: Inconsistent at key]: ", debug[k])
				log.Println("[Tick: value from tree]: ", convertByteToInt(res))
				log.Println("[Tick: value from opi]: ", convertByteToInt(v))
				return false
			}
		}
	}
	return true
}
