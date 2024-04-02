package stateless

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/RiemaLabs/modular-indexer-committee/ord"
	"github.com/RiemaLabs/modular-indexer-committee/ord/getter"
	"github.com/ethereum/go-verkle"

	uint256 "github.com/holiman/uint256"
	"golang.org/x/crypto/sha3"
)

// LocationID is used to indicate the last digit of the Key to fully utilize the characteristics of the Verkle Tree and save memory.
type LocationID = byte

type Dynamic = string

// Tick+Pkscript State
// Key: Keccak256(tick + Pkscript + "static")[:StemSize] + LocationID
// Value: uint256
var AvailableBalancePkscript LocationID = 0x00
var OverallBalancePkscript LocationID = 0x01

func GetTickPkscriptHash(tick string, Pkscript ord.Pkscript, stateID LocationID) []byte {
	tickBytes := []byte(tick)
	uniqueBytes := []byte(Pkscript)
	typeBytes := []byte("static")
	preImg := append(append(uniqueBytes, tickBytes...), typeBytes...)
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(preImg)
	resHash := hasher.Sum(nil)
	return append(resHash[:verkle.StemSize], stateID)
}

func updateBalance(f func(*uint256.Int) *uint256.Int, state KVStorage, tick string, Pkscript ord.Pkscript, loc LocationID) {
	key := GetTickPkscriptHash(tick, Pkscript, loc)
	value := state.GetUInt256(key)
	res := f(value)
	state.InsertUInt256(key, res)
}

// Available, OverallBalances
func GetBalances(state KVStorage, tick string, Pkscript ord.Pkscript) ([]byte, []byte, *uint256.Int, *uint256.Int) {
	key0 := GetTickPkscriptHash(tick, Pkscript, AvailableBalancePkscript)
	key1 := GetTickPkscriptHash(tick, Pkscript, OverallBalancePkscript)
	value0 := state.GetUInt256(key0)
	value1 := state.GetUInt256(key1)
	return key0, key1, value0, value1
}

// Tick State
// Key: Keccak256(tick + "static")[:StemSize] + LocationID
// Value: uint256
var Exists LocationID = 0x00
var RemainingSupply LocationID = 0x01
var MaxSupply LocationID = 0x02
var LimitPerMint LocationID = 0x03
var Decimals LocationID = 0x04

func GetTickHash(tick string, locationID LocationID) []byte {
	tickBytes := []byte(tick)
	typeBytes := []byte("static")
	preImg := append(tickBytes, typeBytes...)
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(preImg)
	resHash := hasher.Sum(nil)
	return append(resHash[:verkle.StemSize], locationID)
}

func GetDecimals(state KVStorage, tick string) ([]byte, *uint256.Int) {
	key := GetTickHash(tick, Decimals)
	value := state.GetUInt256(key)
	return key, value
}

func getTickStatus(tick string) ([]byte, []byte, []byte, []byte, []byte) {
	return GetTickHash(tick, Exists), GetTickHash(tick, RemainingSupply), GetTickHash(tick, MaxSupply), GetTickHash(tick, LimitPerMint), GetTickHash(tick, Decimals)
}

func updateTickState(f func(*uint256.Int) *uint256.Int, state KVStorage, tick string, loc LocationID) {
	key := GetTickHash(tick, loc)
	value := state.GetUInt256(key)
	res := f(value)
	state.InsertUInt256(key, res)
}

// Wallet State
// Key: Keccak256(wallet + "static")[:StemSize] + LocationID
// Value: []byte (Less than 1534 bytes)
var WalletLatestPkscript LocationID = 0x00

func GetWalletHash(wallet string, locationID LocationID) []byte {
	walletBytes := []byte(wallet)
	typeBytes := []byte("static")
	preImg := append(walletBytes, typeBytes...)
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(preImg)
	resHash := hasher.Sum(nil)
	return append(resHash[:verkle.StemSize], locationID)
}

func updateLatestPkscript(state KVStorage, wallet ord.Wallet, Pkscript ord.Pkscript) {
	key := GetWalletHash(string(wallet), WalletLatestPkscript)
	value := string(Pkscript)
	bytes, err := hex.DecodeString(value)
	if err != nil {
		panic(fmt.Errorf("error decoding Pkscript: %v", err))
	}
	state.InsertBytes(key, bytes)
}

func GetLatestPkscript(state KVStorage, wallet string) ([]byte, string) {
	key := GetWalletHash(wallet, WalletLatestPkscript)
	value := state.GetBytes(key)
	return key, hex.EncodeToString(value)
}

// TODO: High. Flush to the disk.
// Inscription Event State
// Key: Keccak256(inscriptionID + "static")[:StemSize] + LocationID
// Value: uint256
var TransferInscribeCount LocationID = 0x0
var TransferTransferCount LocationID = 0x1

// Value: []byte (Less than 34 bytes, Next slot is 0x2 + 1 + (34 / 32) + 1 = 0x5)
var TransferInscribeSourceWallet LocationID = 0x2

// Value: []byte (Less than 1534 bytes)
var TransferInscribeSourcePkscript LocationID = 0x5

func getEventHash(inscriptionID string, locationID LocationID) []byte {
	inscribeBytes := []byte(inscriptionID)
	typeBytes := []byte("static")
	preImg := append(inscribeBytes, typeBytes...)
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(preImg)
	resHash := hasher.Sum(nil)
	return append(resHash[:verkle.StemSize], locationID)
}

func updateWalletAndPkscript(state KVStorage, inscriptionID string, wallet ord.Wallet, Pkscript ord.Pkscript) {
	walletKey := getEventHash(inscriptionID, TransferInscribeSourceWallet)
	walletBytes := decodeBitcoinWallet(string(wallet))
	state.InsertBytes(walletKey, walletBytes)

	PkscriptKey := getEventHash(inscriptionID, TransferInscribeSourcePkscript)
	PkscriptBytes, err := hex.DecodeString(string(Pkscript))
	if err != nil {
		panic(err)
	}
	state.InsertBytes(PkscriptKey, PkscriptBytes)
}

func getWalletAndPkscript(state KVStorage, inscriptionID string) (ord.Wallet, ord.Pkscript) {
	walletKey := getEventHash(inscriptionID, TransferInscribeSourceWallet)
	walletBytes := state.GetBytes(walletKey)
	wallet := encodeBitcoinWallet(walletBytes)
	PkscriptKey := getEventHash(inscriptionID, TransferInscribeSourcePkscript)
	PkscriptBytes := state.GetBytes(PkscriptKey)
	Pkscript := hex.EncodeToString(PkscriptBytes)
	return ord.Wallet(wallet), ord.Pkscript(Pkscript)
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
	state.InsertUInt256(keyExists, uint256.NewInt(1))
	state.InsertUInt256(keyRemainingSupply, maxSupply)
	state.InsertUInt256(keyMaxSupply, maxSupply)
	state.InsertUInt256(keyDecimals, decimals)
	state.InsertUInt256(keyLimitPerMint, limitPerMint)
}

func mintInscribe(state KVStorage, newPkscript ord.Pkscript, newWallet ord.Wallet, tick string, amount *uint256.Int) {
	// update balances
	f_add := func(v *uint256.Int) *uint256.Int {
		return uint256.NewInt(0).Add(v, amount)
	}

	updateBalance(f_add, state, tick, newPkscript, AvailableBalancePkscript)
	updateBalance(f_add, state, tick, newPkscript, OverallBalancePkscript)

	f_sub := func(v *uint256.Int) *uint256.Int {
		return uint256.NewInt(0).Sub(v, amount)
	}
	updateTickState(f_sub, state, tick, RemainingSupply)
	updateLatestPkscript(state, newWallet, newPkscript)
}

func transferInscribe(state KVStorage, inscriptionID string, sourcePkscript ord.Pkscript, sourceWallet ord.Wallet, tick string, amount *uint256.Int) {
	f_sub := func(v *uint256.Int) *uint256.Int {
		return uint256.NewInt(0).Sub(v, amount)
	}
	updateBalance(f_sub, state, tick, sourcePkscript, AvailableBalancePkscript)
	updateLatestPkscript(state, sourceWallet, sourcePkscript)

	// store transfer-inscribe event
	updateWalletAndPkscript(state, inscriptionID, sourceWallet, sourcePkscript)

	// update transfer-inscribe event count
	key := getEventHash(inscriptionID, TransferInscribeCount)
	newEventCount := uint256.NewInt(0).Add(state.GetUInt256(key), uint256.NewInt(1))
	state.InsertUInt256(key, newEventCount)
}

func transferTransferSpendToFee(state KVStorage, inscriptionID string, tick string, amount *uint256.Int) {
	sourceWallet, sourcePkscript := getWalletAndPkscript(state, inscriptionID)
	f_add := func(v *uint256.Int) *uint256.Int {
		return uint256.NewInt(0).Add(v, amount)
	}
	updateBalance(f_add, state, tick, sourcePkscript, AvailableBalancePkscript)
	updateLatestPkscript(state, sourceWallet, sourcePkscript)

	// update transfer-transfer event count
	key := getEventHash(inscriptionID, TransferTransferCount)
	newEventCount := uint256.NewInt(0).Add(state.GetUInt256(key), uint256.NewInt(1))
	state.InsertUInt256(key, newEventCount)
}

func transferTransferNormal(state KVStorage, inscriptionID string, spentPkscript ord.Pkscript, spentWallet ord.Wallet, tick string, amount *uint256.Int) {
	sourceWallet, sourcePkscript := getWalletAndPkscript(state, inscriptionID)
	f_sub := func(v *uint256.Int) *uint256.Int {
		return uint256.NewInt(0).Sub(v, amount)
	}
	updateBalance(f_sub, state, tick, sourcePkscript, OverallBalancePkscript)

	// Don't worry about sourcePkscript == spentPkscript.
	// The update read the value from the storage again.
	f_add := func(v *uint256.Int) *uint256.Int {
		return uint256.NewInt(0).Add(v, amount)
	}
	updateBalance(f_add, state, tick, spentPkscript, AvailableBalancePkscript)
	updateBalance(f_add, state, tick, spentPkscript, OverallBalancePkscript)
	updateLatestPkscript(state, sourceWallet, sourcePkscript)
	updateLatestPkscript(state, spentWallet, spentPkscript)

	// update transfer-transfer event count
	key := getEventHash(inscriptionID, TransferTransferCount)
	newEventCount := uint256.NewInt(0).Add(state.GetUInt256(key), uint256.NewInt(1))
	state.InsertUInt256(key, newEventCount)
}

// TODO: High. Include burn logic.
// Input previous verkle tree and all ord records in a block, then get the K-V array that the verkle tree should update
func Exec(state KVStorage, ots []getter.OrdTransfer, blockHeight uint) {
	if state.GetHeight() != blockHeight-1 {
		panic(fmt.Errorf("mismatched state header: %d and block height: %d", state.GetHeight(), blockHeight-1))
	}
	upperLimit := getLimit()
	if len(ots) == 0 {
		return
	}
	for _, ot := range ots {
		inscriptionID, oldSatpoint, newPkscript, newWallet, sentAsFee, content, contentType :=
			ot.InscriptionID, ot.OldSatpoint, ot.NewPkscript, ot.NewWallet, ot.SentAsFee, ot.Content, ot.ContentType
		var js map[string]string
		json.Unmarshal(content, &js)
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
			// Note: The implementation of upper-lower case conversion for Greek characters differs between Go and Python.
			// Go is employed by us while Python is employed by OPI.
			// Example: tick == "μσ".
			maxSupplyValue, ok := js["max"]
			if !ok {
				continue // invalid inscription
			}
			keyExists, _, _, _, _ := getTickStatus(tick)
			tickExists := state.GetUInt256(keyExists)
			if !tickExists.Eq(uint256.NewInt(0)) {
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
			mintInscribe(state, newPkscript, newWallet, tick, amount)
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
				availableBalance := state.GetUInt256(GetTickPkscriptHash(tick, newPkscript, AvailableBalancePkscript))

				if availableBalance.Lt(amount) {
					continue // not enough available balance
				} else {
					transferInscribe(state, inscriptionID, newPkscript, newWallet, tick, amount)
				}
			} else {
				if isUsedOrInvalid(state, inscriptionID) {
					continue // already used or invalid
				}
				if sentAsFee {
					transferTransferSpendToFee(state, inscriptionID, tick, amount)
				} else {
					transferTransferNormal(state, inscriptionID, newPkscript, newWallet, tick, amount)
				}
			}
		}
	}
}
