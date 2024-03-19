package stateless

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"

	"github.com/RiemaLabs/indexer-committee/ord"
	"github.com/RiemaLabs/indexer-committee/ord/getter"
	"github.com/ethereum/go-verkle"

	uint256 "github.com/holiman/uint256"
	"golang.org/x/crypto/sha3"
)

// LocationID is used to indicate the last digit of the Key to fully utilize the characteristics of the Verkle Tree and save memory.
type LocationID = byte

type Dynamic = string

// Tick+PkScript State
// Key: Keccak256(tick + PkScript + "static")[:StemSize] + LocationID
// Value: uint256
var AvailableBalancePkScript LocationID = 0x00
var OverallBalancePkScript LocationID = 0x01

func getTickPkScriptHash(tick string, PkScript ord.PkScript, stateID LocationID) []byte {
	tickBytes := []byte(tick)
	uniqueBytes := []byte(PkScript)
	typeBytes := []byte("static")
	preImg := append(append(uniqueBytes, tickBytes...), typeBytes...)
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(preImg)
	resHash := hasher.Sum(nil)
	return append(resHash[:verkle.StemSize], stateID)
}

func updateBalance(f func(*uint256.Int) *uint256.Int, state KVStorage, tick string, PkScript ord.PkScript, loc LocationID) {
	key := getTickPkScriptHash(tick, PkScript, loc)
	value := state.GetUInt256(key)
	res := f(value)
	state.InsertUInt256(key, res)
}

// Available, OverallBalances
func GetBalances(state KVStorage, tick string, PkScript ord.PkScript) (*uint256.Int, *uint256.Int) {
	key0 := getTickPkScriptHash(tick, PkScript, AvailableBalancePkScript)
	key1 := getTickPkScriptHash(tick, PkScript, OverallBalancePkScript)
	value0 := state.GetUInt256(key0)
	value1 := state.GetUInt256(key1)
	return value0, value1
}

// Tick State
// Key: Keccak256(tick + "static")[:StemSize] + LocationID
// Value: uint256
var Exists LocationID = 0x00
var RemainingSupply LocationID = 0x01
var MaxSupply LocationID = 0x02
var LimitPerMint LocationID = 0x03
var Decimals LocationID = 0x04

func getTickHash(tick string, locationID LocationID) []byte {
	tickBytes := []byte(tick)
	typeBytes := []byte("static")
	preImg := append(tickBytes, typeBytes...)
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(preImg)
	resHash := hasher.Sum(nil)
	return append(resHash[:verkle.StemSize], locationID)
}

func getTickStatus(tick string) ([]byte, []byte, []byte, []byte, []byte) {
	return getTickHash(tick, Exists), getTickHash(tick, RemainingSupply), getTickHash(tick, MaxSupply), getTickHash(tick, LimitPerMint), getTickHash(tick, Decimals)
}

func updateTickState(f func(*uint256.Int) *uint256.Int, state KVStorage, tick string, loc LocationID) {
	key := getTickHash(tick, loc)
	value := state.GetUInt256(key)
	res := f(value)
	state.InsertUInt256(key, res)
}

// Wallet State
// Key: Keccak256(wallet + "static")[:StemSize] + LocationID
// Value: []byte (Less than 1534 bytes)
var WalletLatestPkScript LocationID = 0x00

func getWalletHash(wallet string, locationID LocationID) []byte {
	walletBytes := []byte(wallet)
	typeBytes := []byte("static")
	preImg := append(walletBytes, typeBytes...)
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(preImg)
	resHash := hasher.Sum(nil)
	return append(resHash[:verkle.StemSize], locationID)
}

func updateLatestPkScript(state KVStorage, wallet ord.Wallet, PkScript ord.PkScript) {
	key := getWalletHash(string(wallet), WalletLatestPkScript)
	value := string(PkScript)
	bytes, err := hex.DecodeString(value)
	if err != nil {
		panic(fmt.Errorf("error decoding PkScript: %v", err))
	}
	state.InsertBytes(key, bytes)
}

func GetLastestPkScript(state KVStorage, wallet string) string {
	key := getWalletHash(wallet, WalletLatestPkScript)
	value := state.GetBytes(key)
	return hex.EncodeToString(value)
}

// TODO: Urgent. Flush to the disk.
// Inscription Event State
// Key: Keccak256(inscriptionID + "static")[:StemSize] + LocationID
// Value: uint256
var TransferInscribeCount LocationID = 0x0
var TransferTransferCount LocationID = 0x1

// Value: []byte (Less than 34 bytes, Next slot is 0x2 + 1 + (34 / 32) + 1 = 0x5)
var TransferInscribeSourceWallet LocationID = 0x2

// Value: []byte (Less than 1534 bytes)
var TransferInscribeSourcePkScript LocationID = 0x5

func getEventHash(inscriptionID string, locationID LocationID) []byte {
	inscribeBytes := []byte(inscriptionID)
	typeBytes := []byte("static")
	preImg := append(inscribeBytes, typeBytes...)
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(preImg)
	resHash := hasher.Sum(nil)
	return append(resHash[:verkle.StemSize], locationID)
}

func updateWalletAndPkScript(state KVStorage, inscriptionID string, wallet ord.Wallet, PkScript ord.PkScript) {
	walletKey := getEventHash(inscriptionID, TransferInscribeSourceWallet)
	walletBytes := decodeBitcoinWallet(string(wallet))
	state.InsertBytes(walletKey, walletBytes)

	PkScriptKey := getEventHash(inscriptionID, TransferInscribeSourcePkScript)
	PkScriptBytes, err := hex.DecodeString(string(PkScript))
	if err != nil {
		panic(err)
	}
	state.InsertBytes(PkScriptKey, PkScriptBytes)
}

func getWalletAndPkScript(state KVStorage, inscriptionID string) (ord.Wallet, ord.PkScript) {
	walletKey := getEventHash(inscriptionID, TransferInscribeSourceWallet)
	walletBytes := state.GetBytes(walletKey)
	wallet := encodeBitcoinWallet(walletBytes)
	PkScriptKey := getEventHash(inscriptionID, TransferInscribeSourcePkScript)
	PkScriptBytes := state.GetBytes(PkScriptKey)
	PkScript := hex.EncodeToString(PkScriptBytes)
	return ord.Wallet(wallet), ord.PkScript(PkScript)
}

func getEventCounts(state KVStorage, inscriptionID string) (*uint256.Int, *uint256.Int) {
	key0 := getEventHash(inscriptionID, TransferInscribeCount)
	key1 := getEventHash(inscriptionID, TransferTransferCount)
	value0 := state.GetUInt256(key0)
	value1 := state.GetUInt256(key1)
	return value0, value1
}

// BRC-20 Computation
func isUsedOrInvalid(state KVStorage, inscriptionID string) bool {
	transferInscribeCount, transferTransferCount := getEventCounts(state, inscriptionID)
	return !transferInscribeCount.Eq(uint256.NewInt(1)) || !transferTransferCount.Eq(uint256.NewInt(0))
}

func deployInscribe(state KVStorage, tick string, maxSupply *uint256.Int, decimals *uint256.Int, limitPerMint *uint256.Int) {
	keyExists, keyRemainingSupply, keyMaxSupply, keyLimitPerMint, keyDecimals := getTickStatus(tick)
	state.InsertUInt256(keyExists, uint256.NewInt(0))
	state.InsertUInt256(keyRemainingSupply, maxSupply)
	state.InsertUInt256(keyMaxSupply, maxSupply)
	state.InsertUInt256(keyLimitPerMint, limitPerMint)
	state.InsertUInt256(keyDecimals, decimals)
}

func mintInscribe(state KVStorage, newPkScript ord.PkScript, newWallet ord.Wallet, tick string, amount *uint256.Int) {
	// update balances
	f_add := func(v *uint256.Int) *uint256.Int {
		return uint256.NewInt(0).Add(v, amount)
	}

	updateBalance(f_add, state, tick, newPkScript, AvailableBalancePkScript)
	updateBalance(f_add, state, tick, newPkScript, OverallBalancePkScript)

	f_sub := func(v *uint256.Int) *uint256.Int {
		return uint256.NewInt(0).Sub(v, amount)
	}
	updateTickState(f_sub, state, tick, RemainingSupply)
	updateLatestPkScript(state, newWallet, newPkScript)
}

func transferInscribe(state KVStorage, inscriptionID string, sourcePkScript ord.PkScript, sourceWallet ord.Wallet, tick string, amount *uint256.Int) {
	f_sub := func(v *uint256.Int) *uint256.Int {
		return uint256.NewInt(0).Sub(v, amount)
	}
	updateBalance(f_sub, state, tick, sourcePkScript, AvailableBalancePkScript)
	updateLatestPkScript(state, sourceWallet, sourcePkScript)

	// store transfer-inscribe event
	updateWalletAndPkScript(state, inscriptionID, sourceWallet, sourcePkScript)

	// update transfer-inscribe event count
	key := getEventHash(inscriptionID, TransferInscribeCount)
	newEventCount := uint256.NewInt(0).Add(state.GetUInt256(key), uint256.NewInt(1))
	state.InsertUInt256(key, newEventCount)
}

func transferTransferSpendToFee(state KVStorage, inscriptionID string, tick string, amount *uint256.Int) {
	sourceWallet, sourcePkScript := getWalletAndPkScript(state, inscriptionID)
	f_add := func(v *uint256.Int) *uint256.Int {
		return uint256.NewInt(0).Add(v, amount)
	}
	updateBalance(f_add, state, tick, sourcePkScript, AvailableBalancePkScript)
	updateLatestPkScript(state, sourceWallet, sourcePkScript)

	// update transfer-transfer event count
	key := getEventHash(inscriptionID, TransferTransferCount)
	newEventCount := uint256.NewInt(0).Add(state.GetUInt256(key), uint256.NewInt(1))
	state.InsertUInt256(key, newEventCount)
}

func transferTransferNormal(state KVStorage, inscriptionID string, spentPkScript ord.PkScript, spentWallet ord.Wallet, tick string, amount *uint256.Int) {
	sourceWallet, sourcePkScript := getWalletAndPkScript(state, inscriptionID)
	f_sub := func(v *uint256.Int) *uint256.Int {
		return uint256.NewInt(0).Sub(v, amount)
	}
	updateBalance(f_sub, state, tick, sourcePkScript, OverallBalancePkScript)

	// Don't worry about sourcePkScript == spentPkScript.
	// The update read the value from the storage again.
	f_add := func(v *uint256.Int) *uint256.Int {
		return uint256.NewInt(0).Add(v, amount)
	}
	updateBalance(f_add, state, tick, spentPkScript, AvailableBalancePkScript)
	updateBalance(f_add, state, tick, spentPkScript, OverallBalancePkScript)
	updateLatestPkScript(state, sourceWallet, sourcePkScript)
	updateLatestPkScript(state, spentWallet, spentPkScript)

	// update transfer-transfer event count
	key := getEventHash(inscriptionID, TransferTransferCount)
	newEventCount := uint256.NewInt(0).Add(state.GetUInt256(key), uint256.NewInt(1))
	state.InsertUInt256(key, newEventCount)
}

func ValidBRC20Transfer(ot getter.OrdTransfer, js map[string]string) (bool, string) {
	oldSatpoint, _, _, sentAsFee, _, contentType :=
		ot.OldSatpoint, ot.NewPkScript, ot.NewWallet, ot.SentAsFee, ot.Content, ot.ContentType
	if sentAsFee && oldSatpoint == "" {
		return false, ""
	}
	if contentType == "" {
		return false, ""
	}
	decodedBytes, err := hex.DecodeString(contentType)
	if err == nil {
		contentType = string(decodedBytes)
	}
	contentType = strings.Split(contentType, ";")[0]
	if contentType != "application/json" && contentType != "text/plain" {
		return false, ""
	}
	tick, ok := js["tick"]
	if !ok {
		return false, ""
	}
	if _, ok := js["op"]; !ok {
		return false, ""
	}
	tick = strings.ToLower(tick)
	// NOTATION1 different to BRC20
	if len(tick) != 4 {
		return false, ""
	}
	return true, tick
}

// Input previous verkle tree and all ord records in a block, then get the K-V array that the verkle tree should update
func Exec(state KVStorage, ots []getter.OrdTransfer) {
	upperLimit := getLimit()
	if len(ots) == 0 {
		return
	}
	for _, ot := range ots {
		inscriptionID, oldSatpoint, newPkScript, newWallet, sentAsFee, content :=
			ot.InscriptionID, ot.OldSatpoint, ot.NewPkScript, ot.NewWallet, ot.SentAsFee, ot.Content
		var js map[string]string
		json.Unmarshal(content, &js)
		valid, tick := ValidBRC20Transfer(ot, js)
		if !valid {
			continue
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
			keyExists, _, _, _, _ := getTickStatus(tick)
			tickExists := state.GetUInt256(keyExists)
			if tickExists.Eq(uint256.NewInt(0)) {
				continue // not deployed
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
				maxSupply, err := getNumberExtendedTo18Decimals(maxSupplyValue, decimals, false)
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
					limitPerMint, err := getNumberExtendedTo18Decimals(lim, decimals, false)
					if err != nil || limitPerMint == nil {
						continue // invalid limit per mint
					}
					if limitPerMint.Gt(upperLimit) || limitPerMint.IsZero() {
						continue // invalid limit per mint
					}
				}
			}
			deployInscribe(state, tick, maxSupply, decimals, limitPerMint)
		}

		// handle mint
		if js["op"] == "mint" && oldSatpoint == "" {
			amountString, ok := js["amt"]
			if !ok {
				continue // invalid inscription
			}
			keyExists, keyRemainingSupply, _, keyLimitPerMint, keyDecimals := getTickStatus(tick)
			tickExists := state.GetUInt256(keyExists)
			if tickExists.Eq(uint256.NewInt(0)) {
				continue // not deployed
			}
			remainingSupply := state.GetUInt256(keyRemainingSupply)
			limitPerMint := state.GetUInt256(keyLimitPerMint)
			decimals := state.GetUInt256(keyDecimals)
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
			mintInscribe(state, newPkScript, newWallet, tick, amount)
		}

		// handle transfer
		if js["op"] == "transfer" {
			amountString, ok := js["amt"]
			if !ok {
				continue // invalid inscription
			}
			keyExists, _, _, _, keyDecimals := getTickStatus(tick)
			tickExists := state.GetUInt256(keyExists)
			if tickExists.Eq(uint256.NewInt(0)) {
				continue // not deployed
			}
			deicmals := state.GetUInt256(keyDecimals)
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
				availableBalance := state.GetUInt256(getTickPkScriptHash(tick, newPkScript, AvailableBalancePkScript))

				if availableBalance.Lt(amount) {
					continue // not enough available balance
				} else {
					transferInscribe(state, inscriptionID, newPkScript, newWallet, tick, amount)
				}
			} else {
				if isUsedOrInvalid(state, inscriptionID) {
					continue // already used or invalid
				}
				if sentAsFee {
					transferTransferSpendToFee(state, inscriptionID, tick, amount)
				} else {
					transferTransferNormal(state, inscriptionID, newPkScript, newWallet, tick, amount)
				}
			}
		}
	}
}
