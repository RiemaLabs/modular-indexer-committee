package ord

import (
	"encoding/hex"
	"encoding/json"
	"log"
	"math/big"
	"strconv"
	"strings"

	"nubit-indexer-committee/internal/ord/getter"

	uint256 "github.com/holiman/uint256"
	"golang.org/x/crypto/sha3"
)

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

func getTickHash(tick string) ([]byte, []byte, []byte, []byte) {
	return getHash("", tick, "tick-exists"), getHash("", tick, "remaining-supply"), getHash("", tick, "limit-per-mint"), getHash("", tick, "decimals")
}

func getEventHash(eventType string, inscrId string) []byte {
	// eventData := []byte(eventType)
	// hasher := sha3.NewLegacyKeccak256()
	// hasher.Write(eventData)
	// tickHash := hasher.Sum(nil)
	// hasher.Write(append([]byte(eventType), tickHash...))
	// return hasher.Sum(nil)
	return getHash("", eventType, inscrId)
}

func deployInscribe(state State, inscrId string, newPkscript string, newAddr string, tick string, maxSupply *uint256.Int, decimals *uint256.Int, limitPerMint *uint256.Int) State {
	keyTick, keyRS, keyLPM, keyD := getTickHash(tick)
	state.Insert(keyTick, convertIntToByte(uint256.NewInt(0)), nodeResolveFn)
	state.Insert(keyRS, convertIntToByte(maxSupply), nodeResolveFn)
	state.Insert(keyLPM, convertIntToByte(limitPerMint), nodeResolveFn)
	state.Insert(keyD, convertIntToByte(decimals), nodeResolveFn)
	return state
}

func mintInscribe(state State, inscrId string, newPkscript string, newAddr string, tick string, amount *uint256.Int) State {
	newAddrByte, _ := decodeBitcoinAddress(newAddr)
	newAddr = string(newAddrByte)

	// store tick + pkscript
	availableKey, overallKey := getHash("available-balance", tick, newPkscript), getHash("overall-balance", tick, newPkscript)
	prevAvailableBalance, prevOverallBalance := state.GetValueOrZero(availableKey), state.GetValueOrZero(overallKey)
	newAvailableBalance, newOverallBalance := uint256.NewInt(0).Add(prevAvailableBalance, amount), uint256.NewInt(0).Add(prevOverallBalance, amount)
	state.Insert(availableKey, convertIntToByte(newAvailableBalance), nodeResolveFn)
	state.Insert(overallKey, convertIntToByte(newOverallBalance), nodeResolveFn)

	// store tick + wallet
	availableKey, overallKey = getHash("available-balance", tick, newAddr), getHash("overall-balance", tick, newAddr)
	state.Insert(availableKey, convertIntToByte(newAvailableBalance), nodeResolveFn)
	state.Insert(overallKey, convertIntToByte(newOverallBalance), nodeResolveFn)

	// update tick info
	_, keyRS, _, _ := getTickHash(tick)
	prevRemainingSupply, _ := state.Get(keyRS, nodeResolveFn)
	newRemainingSupply := uint256.NewInt(0).Sub(convertByteToInt(prevRemainingSupply), amount)
	state.Insert(keyRS, convertIntToByte(newRemainingSupply), nodeResolveFn)
	return state
}

// save decoded wallet address and pkscript
func saveSourceWalletAndPkscript(state State, inscrId string, sourceAddr string, pkScript string) {
	eventKey := getEventHash("transfer-inscribe-source-wallet", inscrId)
	state.Insert(eventKey, []byte(sourceAddr), nodeResolveFn)

	length := len(pkScript)
	prefix := []byte{byte(length)}
	if len(pkScript)%2 == 1 {
		pkScript += "0"
	}
	encodedPkscript, _ := hex.DecodeString(pkScript)
	encodedPkscript = append(prefix, encodedPkscript...)
	pkScriptKey1 := getEventHash("transfer-inscribe-source-pkscript-1", inscrId)
	b1, _ := padTo32Bytes(encodedPkscript[:min(len(encodedPkscript), 32)])
	state.Insert(pkScriptKey1, b1, nodeResolveFn)
	if len(encodedPkscript) > 32 {
		pkScriptKey2 := getEventHash("transfer-inscribe-source-pkscript-2", inscrId)
		b2, _ := padTo32Bytes(encodedPkscript[32:])
		state.Insert(pkScriptKey2, b2, nodeResolveFn)
	}
}

// get decoded wallet address and pkscript
func getSourceWalletAndPkscript(state State, inscrId string) (string, string) {
	eventKey := getEventHash("transfer-inscribe-source-wallet", inscrId)
	sourceAddr, _ := state.Get(eventKey, nodeResolveFn)

	pkScriptKey1, pkScriptKey2 := getEventHash("transfer-inscribe-source-pkscript-1", inscrId), getEventHash("transfer-inscribe-source-pkscript-2", inscrId)
	b1, _ := state.Get(pkScriptKey1, nodeResolveFn)
	b2, _ := state.Get(pkScriptKey2, nodeResolveFn)
	b := append(b1, b2...)
	length := int(b[0])
	sourcePkscript := hex.EncodeToString(b[1:])[:length]
	return string(sourceAddr), sourcePkscript
}

func transferInscribe(state State, inscrId string, sourcePkScript string, sourceAddr string, tick string, amount *uint256.Int, availableBalance *uint256.Int) State {
	sourceAddrByte, _ := decodeBitcoinAddress(sourceAddr)
	sourceAddr = string(sourceAddrByte)

	newAvailableBalance := uint256.NewInt(0).Sub(availableBalance, amount)
	availableKey := getHash("available-balance", tick, sourceAddr)
	state.Insert(availableKey, convertIntToByte(newAvailableBalance), nodeResolveFn)
	availableKey = getHash("available-balance", tick, sourcePkScript)
	state.Insert(availableKey, convertIntToByte(newAvailableBalance), nodeResolveFn)

	// store transfer-inscribe event
	saveSourceWalletAndPkscript(state, inscrId, sourceAddr, sourcePkScript)

	// update transfer-inscribe event count
	eventCntKey := getEventHash("transfer-inscribe-count", inscrId)
	newEventCnt := uint256.NewInt(0).Add(state.GetValueOrZero(eventCntKey), uint256.NewInt(1))
	state.Insert(eventCntKey, convertIntToByte(newEventCnt), nodeResolveFn)

	return state
}

func isUsedOrInvalid(state State, inscrId string) bool {
	tIEventKey := getEventHash("transfer-inscribe-count", inscrId)
	transferInscribeCnt := state.GetValueOrZero(tIEventKey)

	tTEventKey := getEventHash("transfer-transfer-count", inscrId)
	transferTransferCnt := state.GetValueOrZero(tTEventKey)

	return !transferInscribeCnt.Eq(uint256.NewInt(1)) || !transferTransferCnt.IsZero()
}

func transferTransferSpendToFee(state State, inscrId string, tick string, amount *uint256.Int, txId uint) State {
	sourceAddr, sourcePkScript := getSourceWalletAndPkscript(state, inscrId)
	availableKey := getHash("available-balance", tick, sourceAddr)
	lastAvailableBalance := state.GetValueOrZero(availableKey)
	newAvailableBalance := uint256.NewInt(0).Add(lastAvailableBalance, amount)
	state.Insert(availableKey, convertIntToByte(newAvailableBalance), nodeResolveFn)
	availableKey = getHash("available-balance", tick, sourcePkScript)
	state.Insert(availableKey, convertIntToByte(newAvailableBalance), nodeResolveFn)

	// update transfer-transfer event count
	eventCntKey := getEventHash("transfer-transfer-count", inscrId)
	newTransferTransferCnt := uint256.NewInt(0).Add(state.GetValueOrZero(eventCntKey), uint256.NewInt(1))
	state.Insert(eventCntKey, convertIntToByte(newTransferTransferCnt), nodeResolveFn)

	return state
}

func transferTransferNormal(state State, inscrId string, spentPkScript string, spentAddr string, tick string, amount *uint256.Int, txId uint) State {
	spentAddrByte, _ := decodeBitcoinAddress(spentAddr)
	spentAddr = string(spentAddrByte)

	sourceAddr, sourcePkScript := getSourceWalletAndPkscript(state, inscrId)
	sourceOverallKey := getHash("overall-balance", tick, sourceAddr)
	newSourceOverallBalance := uint256.NewInt(0).Sub(state.GetValueOrZero(sourceOverallKey), amount)
	state.Insert(sourceOverallKey, convertIntToByte(newSourceOverallBalance), nodeResolveFn)
	sourceOverallKey = getHash("overall-balance", tick, sourcePkScript)
	state.Insert(sourceOverallKey, convertIntToByte(newSourceOverallBalance), nodeResolveFn)

	spentAvailableKey, spentOverallKey := getHash("available-balance", tick, spentAddr), getHash("overall-balance", tick, spentAddr)
	newSpentAvailableBalance, newSpentOverallBalance := uint256.NewInt(0).Add(state.GetValueOrZero(spentAvailableKey), amount), uint256.NewInt(0).Add(state.GetValueOrZero(spentOverallKey), amount)
	state.Insert(spentAvailableKey, convertIntToByte(newSpentAvailableBalance), nodeResolveFn)
	state.Insert(spentOverallKey, convertIntToByte(newSpentOverallBalance), nodeResolveFn)
	spentAvailableKey, spentOverallKey = getHash("available-balance", tick, spentPkScript), getHash("overall-balance", tick, spentPkScript)
	state.Insert(spentAvailableKey, convertIntToByte(newSpentAvailableBalance), nodeResolveFn)
	state.Insert(spentOverallKey, convertIntToByte(newSpentOverallBalance), nodeResolveFn)

	// update transfer-transfer event count
	eventCntKey := getEventHash("transfer-transfer-count", inscrId)
	newTransferTransferCnt := uint256.NewInt(0).Add(state.GetValueOrZero(eventCntKey), uint256.NewInt(1))
	state.Insert(eventCntKey, convertIntToByte(newTransferTransferCnt), nodeResolveFn)
	return state
}

// Input previous verkle tree and all ord records in a block, then get the K-V array that the verkle tree should update
func Exec(state State, ordTransfer []getter.OrdTransfer) State {
	upperLimit := getLimit()
	if len(ordTransfer) == 0 {
		return state
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
			if v, _ := state.Get(keyTick, nodeResolveFn); len(v) != 0 {
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
			state = deployInscribe(state, inscrId, newPkscript, newAddr, tick, maxSupply, decimals, limitPerMint)
		}

		// handle mint
		if js["op"] == "mint" && oldSatpoint == "" {
			amountString, ok := js["amt"]
			if !ok {
				continue // invalid inscription
			}
			keyTick, keyRS, keyLPM, keyD := getTickHash(tick)
			tickExists, _ := state.Get(keyTick, nodeResolveFn)
			if len(tickExists) == 0 {
				continue // not deployed
			}
			remainingSupplyBytes, _ := state.Get(keyRS, nodeResolveFn)
			limitPerMintBytes, _ := state.Get(keyLPM, nodeResolveFn)
			decimalsBytes, _ := state.Get(keyD, nodeResolveFn)
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
			state = mintInscribe(state, inscrId, newPkscript, newAddr, tick, amount)
		}

		// handle transfer
		if js["op"] == "transfer" {
			amountString, ok := js["amt"]
			if !ok {
				continue // invalid inscription
			}
			keyTick, _, _, keyD := getTickHash(tick)
			tickExists, _ := state.Get(keyTick, nodeResolveFn)
			decimalBytes, _ := state.Get(keyD, nodeResolveFn)
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
				availableBalance := state.GetValueOrZero(getHash("available-balance", tick, newPkscript))

				if availableBalance.Lt(amount) {
					continue // not enough available balance
				} else {
					state = transferInscribe(state, inscrId, newPkscript, newAddr, tick, amount, availableBalance)
				}
			} else {
				if isUsedOrInvalid(state, inscrId) {
					continue // already used or invalid
				}
				if sentAsFee {
					state = transferTransferSpendToFee(state, inscrId, tick, amount, txId)
				} else {
					state = transferTransferNormal(state, inscrId, newPkscript, newAddr, tick, amount, txId)
				}
			}
		}
	}
	return state
}
