package ord

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"

	"github.com/RiemaLabs/indexer-committee/ord/getter"

	uint256 "github.com/holiman/uint256"
	"golang.org/x/crypto/sha3"
)

type StateID = [1]byte

// tick - pkscript - uint256
var AvailableBalancePkscript StateID = StateID{0x0}

// tick - wallet - uint256
var AvailableBalance StateID = StateID{0x1}

// tick - pkscript - uint256
var OverallBalancePkscript StateID = StateID{0x2}

// tick - wallet - uint256
var OverallBalance StateID = StateID{0x3}

// tick - bool
var Exists StateID = StateID{0x4}

// tick - uint256
var RemainingSupply StateID = StateID{0x5}

// tick - uint256
var MaxSupply StateID = StateID{0x6}

// tick - uint256
var LimitPerMint StateID = StateID{0x7}

// tick - uint256
var Decimals StateID = StateID{0x8}

type EventID = [4]byte

// event - TransferInscribeSourceWallet
var TransferInscribeSourceWallet EventID = EventID{0x0}

// event - TransferInscribeSourcePkscript1
var TransferInscribeSourcePkscript1 EventID = EventID{0x1}

// event - TransferInscribeSourcePkscript2
var TransferInscribeSourcePkscript2 EventID = EventID{0x2}

// event - TransferTransferCount
var TransferTransferCount EventID = EventID{0x3}

// event - TransferInscribeCount
var TransferInscribeCount EventID = EventID{0x4}

// Get hash value by keccak224(uniqueID)[:27] (27bytes) + stateID (1bytes) + tick (4bytes).
// Or, get hash value by keccak224(uniqueID)[:26] (26bytes) + stateID (1bytes) + tick (5bytes).
func GetHash(stateID StateID, uniqueID string, tick string) []byte {
	prefix := uniqueID
	prefixBytes := []byte(prefix)
	hasher := sha3.New224()
	hasher.Write(prefixBytes)
	prefixHash := hasher.Sum(nil)
	var res []byte
	if !(len(tick) == 4 || len(tick) == 5) {
		panic(fmt.Sprintf("Tick must be 4 or 5 bytes! Current is %s", tick))
	} else if len(tick) == 4 {
		res = append(append(prefixHash[:27], stateID[:]...), []byte(tick)...)
	} else {
		// Introduced by the BP04: https://github.com/brc20-devs/brc20-proposals/blob/main/bp04-self-mint/proposal.md
		res = append(append(prefixHash[:26], stateID[:]...), []byte(tick)...)
	}
	if len(res) != 32 {
		panic(fmt.Sprintf("Key must be 32 bytes! Current is %d", len(res)))
	}
	return res
}

func GetTickStatus(tick string) ([]byte, []byte, []byte, []byte, []byte) {
	return GetHash(Exists, "", tick), GetHash(RemainingSupply, "", tick), GetHash(MaxSupply, "", tick), GetHash(LimitPerMint, "", tick), GetHash(Decimals, "", tick)
}

// Get hash value by eventID (4bytes) + keccak224(inscrID) (28 bytes).
func GetEventHash(eventID EventID, inscrID string) []byte {
	inscrIDByte := []byte(inscrID)
	hasher := sha3.New224()
	hasher.Write(inscrIDByte)
	inscrIDHash := hasher.Sum(nil)
	return append(eventID[:], inscrIDHash...)
}

func deployInscribe(state State, inscrID string, newPkscript string, newAddr string, tick string, maxSupply *uint256.Int, decimals *uint256.Int, limitPerMint *uint256.Int) State {
	keyExists, keyRemainingSupply, keyMaxSupply, keyLimitPerMint, keyDecimals := GetTickStatus(tick)
	state.Insert(keyExists, convertIntToByte(uint256.NewInt(0)), nodeResolveFn)
	state.Insert(keyRemainingSupply, convertIntToByte(maxSupply), nodeResolveFn)
	state.Insert(keyMaxSupply, convertIntToByte(maxSupply), nodeResolveFn)
	state.Insert(keyLimitPerMint, convertIntToByte(limitPerMint), nodeResolveFn)
	state.Insert(keyDecimals, convertIntToByte(decimals), nodeResolveFn)
	return state
}

func mintInscribe(state State, inscrID string, newPkscript string, newAddr string, tick string, amount *uint256.Int) State {
	newAddrByte, _ := decodeBitcoinAddress(newAddr)
	newAddr = string(newAddrByte)

	// store tick + pkscript
	availableKey, overallKey := GetHash(AvailableBalancePkscript, newPkscript, tick), GetHash(OverallBalancePkscript, newPkscript, tick)
	prevAvailableBalance, prevOverallBalance := state.GetValueOrZero(availableKey), state.GetValueOrZero(overallKey)
	newAvailableBalance, newOverallBalance := uint256.NewInt(0).Add(prevAvailableBalance, amount), uint256.NewInt(0).Add(prevOverallBalance, amount)
	state.Insert(availableKey, convertIntToByte(newAvailableBalance), nodeResolveFn)
	state.Insert(overallKey, convertIntToByte(newOverallBalance), nodeResolveFn)

	// store tick + wallet
	availableKey, overallKey = GetHash(AvailableBalance, newAddr, tick), GetHash(OverallBalance, newAddr, tick)
	state.Insert(availableKey, convertIntToByte(newAvailableBalance), nodeResolveFn)
	state.Insert(overallKey, convertIntToByte(newOverallBalance), nodeResolveFn)

	// update tick info
	_, keyRemainingSupply, _, _, _ := GetTickStatus(tick)
	prevRemainingSupply, _ := state.Get(keyRemainingSupply, nodeResolveFn)
	newRemainingSupply := uint256.NewInt(0).Sub(convertByteToInt(prevRemainingSupply), amount)
	state.Insert(keyRemainingSupply, convertIntToByte(newRemainingSupply), nodeResolveFn)
	return state
}

// save decoded wallet address and pkscript
func saveSourceWalletAndPkscript(state State, inscrID string, sourceAddr string, pkScript string) {
	eventKey := GetEventHash(TransferInscribeSourceWallet, inscrID)
	state.Insert(eventKey, []byte(sourceAddr), nodeResolveFn)

	length := len(pkScript)
	prefix := []byte{byte(length)}
	if len(pkScript)%2 == 1 {
		pkScript += "0"
	}
	encodedPkscript, _ := hex.DecodeString(pkScript)
	encodedPkscript = append(prefix, encodedPkscript...)
	pkScriptKey1 := GetEventHash(TransferInscribeSourcePkscript1, inscrID)
	b1, _ := padTo32Bytes(encodedPkscript[:min(len(encodedPkscript), 32)])
	state.Insert(pkScriptKey1, b1, nodeResolveFn)
	if len(encodedPkscript) > 32 {
		pkScriptKey2 := GetEventHash(TransferInscribeSourcePkscript2, inscrID)
		b2, _ := padTo32Bytes(encodedPkscript[32:])
		state.Insert(pkScriptKey2, b2, nodeResolveFn)
	}
}

// get decoded wallet address and pkscript
func getSourceWalletAndPkscript(state State, inscrID string) (string, string) {
	eventKey := GetEventHash(TransferInscribeSourceWallet, inscrID)
	sourceAddr, _ := state.Get(eventKey, nodeResolveFn)

	pkScriptKey1, pkScriptKey2 := GetEventHash(TransferInscribeSourcePkscript1, inscrID), GetEventHash(TransferInscribeSourcePkscript2, inscrID)
	b1, _ := state.Get(pkScriptKey1, nodeResolveFn)
	b2, _ := state.Get(pkScriptKey2, nodeResolveFn)
	b := append(b1, b2...)
	length := int(b[0])
	sourcePkscript := hex.EncodeToString(b[1:])[:length]
	return string(sourceAddr), sourcePkscript
}

func transferInscribe(state State, inscrID string, sourcePkScript string, sourceAddr string, tick string, amount *uint256.Int, availableBalance *uint256.Int) State {
	sourceAddrByte, _ := decodeBitcoinAddress(sourceAddr)
	sourceAddr = string(sourceAddrByte)

	newAvailableBalance := uint256.NewInt(0).Sub(availableBalance, amount)
	availableKey := GetHash(AvailableBalance, sourceAddr, tick)
	state.Insert(availableKey, convertIntToByte(newAvailableBalance), nodeResolveFn)
	availableKey = GetHash(AvailableBalancePkscript, sourcePkScript, tick)
	state.Insert(availableKey, convertIntToByte(newAvailableBalance), nodeResolveFn)

	// store transfer-inscribe event
	saveSourceWalletAndPkscript(state, inscrID, sourceAddr, sourcePkScript)

	// update transfer-inscribe event count
	eventCntKey := GetEventHash(TransferInscribeCount, inscrID)
	newEventCnt := uint256.NewInt(0).Add(state.GetValueOrZero(eventCntKey), uint256.NewInt(1))
	state.Insert(eventCntKey, convertIntToByte(newEventCnt), nodeResolveFn)

	return state
}

func isUsedOrInvalid(state State, inscrID string) bool {
	tIEventKey := GetEventHash(TransferInscribeCount, inscrID)
	transferInscribeCnt := state.GetValueOrZero(tIEventKey)

	tTEventKey := GetEventHash(TransferTransferCount, inscrID)
	transferTransferCnt := state.GetValueOrZero(tTEventKey)

	return !transferInscribeCnt.Eq(uint256.NewInt(1)) || !transferTransferCnt.IsZero()
}

func transferTransferSpendToFee(state State, inscrID string, tick string, amount *uint256.Int, txId uint) State {
	sourceAddr, sourcePkScript := getSourceWalletAndPkscript(state, inscrID)
	availableKey := GetHash(AvailableBalance, sourceAddr, tick)
	lastAvailableBalance := state.GetValueOrZero(availableKey)
	newAvailableBalance := uint256.NewInt(0).Add(lastAvailableBalance, amount)
	state.Insert(availableKey, convertIntToByte(newAvailableBalance), nodeResolveFn)
	availableKey = GetHash(AvailableBalancePkscript, sourcePkScript, tick)
	state.Insert(availableKey, convertIntToByte(newAvailableBalance), nodeResolveFn)

	// update transfer-transfer event count
	eventCntKey := GetEventHash(TransferTransferCount, inscrID)
	newTransferTransferCnt := uint256.NewInt(0).Add(state.GetValueOrZero(eventCntKey), uint256.NewInt(1))
	state.Insert(eventCntKey, convertIntToByte(newTransferTransferCnt), nodeResolveFn)

	return state
}

func transferTransferNormal(state State, inscrID string, spentPkScript string, spentAddr string, tick string, amount *uint256.Int, txId uint) State {
	spentAddrByte, _ := decodeBitcoinAddress(spentAddr)
	spentAddr = string(spentAddrByte)

	sourceAddr, sourcePkScript := getSourceWalletAndPkscript(state, inscrID)
	sourceOverallKey := GetHash(OverallBalance, sourceAddr, tick)
	newSourceOverallBalance := uint256.NewInt(0).Sub(state.GetValueOrZero(sourceOverallKey), amount)
	state.Insert(sourceOverallKey, convertIntToByte(newSourceOverallBalance), nodeResolveFn)

	sourcePkOverallKey := GetHash(OverallBalancePkscript, sourcePkScript, tick)
	state.Insert(sourcePkOverallKey, convertIntToByte(newSourceOverallBalance), nodeResolveFn)

	spentAvailableKey, spentOverallKey := GetHash(AvailableBalance, spentAddr, tick), GetHash(OverallBalance, spentAddr, tick)
	newSpentAvailableBalance, newSpentOverallBalance := uint256.NewInt(0).Add(state.GetValueOrZero(spentAvailableKey), amount), uint256.NewInt(0).Add(state.GetValueOrZero(spentOverallKey), amount)
	state.Insert(spentAvailableKey, convertIntToByte(newSpentAvailableBalance), nodeResolveFn)
	state.Insert(spentOverallKey, convertIntToByte(newSpentOverallBalance), nodeResolveFn)
	spentAvailableKey, spentOverallKey = GetHash(AvailableBalancePkscript, spentPkScript, tick), GetHash(OverallBalancePkscript, spentPkScript, tick)
	state.Insert(spentAvailableKey, convertIntToByte(newSpentAvailableBalance), nodeResolveFn)
	state.Insert(spentOverallKey, convertIntToByte(newSpentOverallBalance), nodeResolveFn)

	// update transfer-transfer event count
	eventCntKey := GetEventHash(TransferTransferCount, inscrID)
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
		txId, inscrID, oldSatpoint, newPkscript, newAddr, sentAsFee, contentType := transfer.ID, transfer.InscriptionID, transfer.OldSatpoint, transfer.NewPkscript, transfer.NewWallet, transfer.SentAsFee, transfer.ContentType
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
			keyExists, _, _, _, _ := GetTickStatus(tick)
			if v, _ := state.Get(keyExists, nodeResolveFn); len(v) != 0 {
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
			state = deployInscribe(state, inscrID, newPkscript, newAddr, tick, maxSupply, decimals, limitPerMint)
		}

		// handle mint
		if js["op"] == "mint" && oldSatpoint == "" {
			amountString, ok := js["amt"]
			if !ok {
				continue // invalid inscription
			}
			keyExists, keyRemainingSupply, _, keyLimitPerMint, keyDecimals := GetTickStatus(tick)
			tickExists, _ := state.Get(keyExists, nodeResolveFn)
			if len(tickExists) == 0 {
				continue // not deployed
			}
			remainingSupplyBytes, _ := state.Get(keyRemainingSupply, nodeResolveFn)
			limitPerMintBytes, _ := state.Get(keyLimitPerMint, nodeResolveFn)
			decimalsBytes, _ := state.Get(keyDecimals, nodeResolveFn)
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
			state = mintInscribe(state, inscrID, newPkscript, newAddr, tick, amount)
		}

		// handle transfer
		if js["op"] == "transfer" {
			amountString, ok := js["amt"]
			if !ok {
				continue // invalid inscription
			}
			keyExists, _, _, _, keyDecimals := GetTickStatus(tick)
			tickExists, _ := state.Get(keyExists, nodeResolveFn)
			decimalBytes, _ := state.Get(keyDecimals, nodeResolveFn)
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
				availableBalance := state.GetValueOrZero(GetHash(AvailableBalancePkscript, newPkscript, tick))

				if availableBalance.Lt(amount) {
					continue // not enough available balance
				} else {
					state = transferInscribe(state, inscrID, newPkscript, newAddr, tick, amount, availableBalance)
				}
			} else {
				if isUsedOrInvalid(state, inscrID) {
					continue // already used or invalid
				}
				if sentAsFee {
					state = transferTransferSpendToFee(state, inscrID, tick, amount, txId)
				} else {
					state = transferTransferNormal(state, inscrID, newPkscript, newAddr, tick, amount, txId)
				}
			}
		}
	}
	return state
}
