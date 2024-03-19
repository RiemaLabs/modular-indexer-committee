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
	err := state.InsertUInt256(key, res)
	if err != nil {
		panic(err)
	}
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

func GetTickStatus(tick string) ([]byte, []byte, []byte, []byte, []byte) {
	return GetTickHash(tick, Exists), GetTickHash(tick, RemainingSupply), GetTickHash(tick, MaxSupply), GetTickHash(tick, LimitPerMint), GetTickHash(tick, Decimals)
}

func updateTickState(f func(*uint256.Int) *uint256.Int, state KVStorage, tick string, loc LocationID) {
	key := GetTickHash(tick, loc)
	value := state.GetUInt256(key)
	res := f(value)
	err := state.InsertUInt256(key, res)
	if err != nil {
		panic(err)
	}
}

// Wallet State
// Key: Keccak256(wallet + "Pkscript")[:StemSize]
// Value: string
// Use the whole 256 slots to store the string.
var WalletLatestPkscript Dynamic = "Pkscript"

func GetWalletHashForDynamic(wallet string, dynamicString Dynamic) []byte {
	walletBytes := []byte(wallet)
	typeBytes := []byte(dynamicString)
	preImg := append(walletBytes, typeBytes...)
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(preImg)
	resHash := hasher.Sum(nil)
	return append(resHash[:verkle.StemSize], 0x00)
}

func updateLatestPkscript(state KVStorage, wallet string, Pkscript string) {
	key := GetWalletHashForDynamic(wallet, WalletLatestPkscript)
	value := Pkscript
	bytes, err := hex.DecodeString(value)
	if err != nil {
		panic(fmt.Errorf("Error decoding Pkscript: %v", err))
	}
	err = state.InsertBytes(key, bytes)
	if err != nil {
		panic(err)
	}
}

// Setter
func deployInscribe(state KVStorage, tick string, maxSupply *uint256.Int, decimals *uint256.Int, limitPerMint *uint256.Int) {
	keyExists, keyRemainingSupply, keyMaxSupply, keyLimitPerMint, keyDecimals := GetTickStatus(tick)
	err := state.InsertUInt256(keyExists, uint256.NewInt(0))
	if err != nil {
		panic(err)
	}
	err = state.InsertUInt256(keyRemainingSupply, maxSupply)
	if err != nil {
		panic(err)
	}
	err = state.InsertUInt256(keyMaxSupply, maxSupply)
	if err != nil {
		panic(err)
	}
	err = state.InsertUInt256(keyLimitPerMint, limitPerMint)
	if err != nil {
		panic(err)
	}
	err = state.InsertUInt256(keyDecimals, decimals)
	if err != nil {
		panic(err)
	}
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

	// TODO: update the latest wallet-script binding
}

func transferInscribe(state KVStorage, sourcePkscript ord.Pkscript, sourceWallet ord.Wallet, tick string, amount *uint256.Int) {
	f_sub := func(v *uint256.Int) *uint256.Int {
		return uint256.NewInt(0).Sub(v, amount)
	}
	updateBalance(f_sub, state, tick, sourcePkscript, AvailableBalancePkscript)
}

func transferTransferSpendToFee(state KVStorage, sourcePkscript ord.Pkscript, sourceWallet ord.Wallet, tick string, amount *uint256.Int) {
	f_add := func(v *uint256.Int) *uint256.Int {
		return uint256.NewInt(0).Add(v, amount)
	}
	updateBalance(f_add, state, tick, sourcePkscript, AvailableBalancePkscript)
}

func transferTransferNormal(state KVStorage, sourcePkscript ord.Pkscript, sourceWallet ord.Wallet, spentPkscript ord.Pkscript, spentWallet ord.Wallet, tick string, amount *uint256.Int) {
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
}

// Input previous verkle tree and all ord records in a block, then get the K-V array that the verkle tree should update
func Exec(state KVStorage, ordTransfer []getter.OrdTransfer) {
	upperLimit := getLimit()
	if len(ordTransfer) == 0 {
		return
	}
	for _, transfer := range ordTransfer {
		oldSatpoint, newPkscript, newWallet, sentAsFee, contentType :=
			transfer.OldSatpoint, transfer.NewPkscript, transfer.NewWallet, transfer.SentAsFee, transfer.ContentType
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
			keyExists, keyRemainingSupply, _, keyLimitPerMint, keyDecimals := GetTickStatus(tick)
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
			keyExists, _, _, _, keyDecimals := GetTickStatus(tick)
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
					transferInscribe(state, newPkscript, newWallet, tick, amount)
				}
			} else {
				isUsedOrInvalid := func(transferInscribeDone bool, transferTransferDone bool) bool {
					return !transferInscribeDone || transferTransferDone
				}
				if isUsedOrInvalid(transfer.TransferInscribeDone, transfer.TransferTransferDone) {
					continue // already used or invalid
				}
				tWallet, tPkscript := transfer.TransferInscribeWallet, transfer.TransferInscribePkscript
				if sentAsFee {
					transferTransferSpendToFee(state, tPkscript, tWallet, tick, amount)
				} else {
					transferTransferNormal(state, tPkscript, tWallet, newPkscript, newWallet, tick, amount)
				}
			}
		}
	}
}
