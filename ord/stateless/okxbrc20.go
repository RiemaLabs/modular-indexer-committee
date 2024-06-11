package stateless

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-verkle"
	uint256 "github.com/holiman/uint256"
	"golang.org/x/crypto/sha3"

	"github.com/RiemaLabs/modular-indexer-committee/ord"
	"github.com/RiemaLabs/modular-indexer-committee/ord/getter"
)

// LocationID is used to indicate the last digit of the Key to fully utilize the characteristics of the Verkle Tree and save memory.
type LocationID = byte

type Dynamic = string

// Tick+Wallet State
// Key: Keccak256(tick + Wallet + "GetTickWalletHash")[:StemSize] + LocationID
// Value: uint256
var AvailableBalanceWallet LocationID = 0x00
var OverallBalanceWallet LocationID = 0x01

func GetTickWalletHash(tick string, wallet ord.Wallet, stateID LocationID) []byte {
	tickBytes := []byte(tick)
	uniqueBytes := []byte(wallet)
	typeBytes := []byte("GetTickWalletHash")
	preImg := append(append(tickBytes, uniqueBytes...), typeBytes...)
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(preImg)
	resHash := hasher.Sum(nil)
	return append(resHash[:verkle.StemSize], stateID)
}

func updateBalance(f func(*uint256.Int) *uint256.Int, state KVStorage, tick string, wallet ord.Wallet, loc LocationID) {
	key := GetTickWalletHash(tick, wallet, loc)
	value := state.GetUInt256(key)
	res := f(value)
	state.InsertUInt256(key, res)
}

// Available, OverallBalances
func GetBalances(state KVStorage, tick string, wallet ord.Wallet) ([]byte, []byte, *uint256.Int, *uint256.Int) {
	key0 := GetTickWalletHash(tick, wallet, AvailableBalanceWallet)
	key1 := GetTickWalletHash(tick, wallet, OverallBalanceWallet)
	value0 := state.GetUInt256(key0)
	value1 := state.GetUInt256(key1)
	return key0, key1, value0, value1
}

// Tick State
// Key: Keccak256(tick + "GetTickHash")[:StemSize] + LocationID
// Value: uint256
var Exists LocationID = 0x00
var RemainingSupply LocationID = 0x01
var MaxSupply LocationID = 0x02
var LimitPerMint LocationID = 0x03
var Decimals LocationID = 0x04
var IsSelfMint LocationID = 0x05
var InscriptionID LocationID = 0x06 // inscription should take 2 slots, next should start with 08

func GetTickHash(tick string, locationID LocationID) []byte {
	tickBytes := []byte(tick)
	typeBytes := []byte("GetTickHash")
	preImg := append(tickBytes, typeBytes...)
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(preImg)
	resHash := hasher.Sum(nil)
	return append(resHash[:verkle.StemSize], locationID)
}

func getTickStatus(tick string) ([]byte, []byte, []byte, []byte, []byte, []byte, []byte) {
	return GetTickHash(tick, Exists), GetTickHash(tick, RemainingSupply), GetTickHash(tick, MaxSupply), GetTickHash(tick, LimitPerMint), GetTickHash(tick, Decimals), GetTickHash(tick, InscriptionID), GetTickHash(tick, IsSelfMint)
}

func updateTickState(f func(*uint256.Int) *uint256.Int, state KVStorage, tick string, loc LocationID) {
	key := GetTickHash(tick, loc)
	value := state.GetUInt256(key)
	res := f(value)
	state.InsertUInt256(key, res)
}

// TODO: High. Flush to the disk.
// Inscription Event State
// Key: Keccak256(inscriptionID + "GetEventHash")[:StemSize] + LocationID
// Value: uint256
var TransferInscribeCount LocationID = 0x0
var TransferTransferCount LocationID = 0x1

// Value: []byte (Less than 34 bytes, Next slot is 0x2 + 1 + (34 / 32) + 1 = 0x5)
var TransferInscribeSourceWallet LocationID = 0x2

func GetEventHash(inscriptionID string, locationID LocationID) []byte {
	inscribeBytes := []byte(inscriptionID)
	typeBytes := []byte("GetEventHash")
	preImg := append(inscribeBytes, typeBytes...)
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(preImg)
	resHash := hasher.Sum(nil)
	return append(resHash[:verkle.StemSize], locationID)
}

func processDeploy(state KVStorage, deployEvent *getter.BRC20DeployEvent) {
	if !deployEvent.Valid {
		return
	}
	// get tick name
	tick := deployEvent.Tick
	keyExists, _, _, _, _, _, _ := getTickStatus(tick)
	tickExists := state.GetUInt256(keyExists)
	if tickExists.Eq(uint256.NewInt(1)) {
		return // deployed
	}

	inscriptionID := deployEvent.InscriptionID
	upperLimit := getLimit()

	// process decimals
	if deployEvent.Decimal > MaxDecimalWidth || deployEvent.Decimal < 0 {
		return
	}
	decimals, _ := uint256.FromBig(big.NewInt(int64(deployEvent.Decimal)))

	// process maxSupply
	var maxSupply *uint256.Int
	if !isPositiveNumberWithDot(deployEvent.Supply, false) {
		return
	}
	maxSupply, err := get18DecimalNumbers(deployEvent.Supply, decimals, false)
	if err != nil || maxSupply == nil {
		return // invalid max supply
	}
	if maxSupply.Gt(upperLimit) || maxSupply.IsZero() {
		return // invalid max supply
	}

	// process limitPerMint
	limitPerMint := maxSupply
	if !isPositiveNumberWithDot(deployEvent.LimitPerMint, false) {
		return // invalid limit per mint
	}
	limitPerMint, err = get18DecimalNumbers(deployEvent.LimitPerMint, decimals, false)
	if err != nil || limitPerMint == nil {
		return // invalid limit per mint
	}
	if limitPerMint.Gt(upperLimit) || limitPerMint.IsZero() {
		return // invalid limit per mint
	}

	// process self-mint tick
	isSelfMint := false
	if len(tick) == int(SelfMintTickLenght) {
		if deployEvent.BlockHeight < SelfMintEnableHeight {
			return
		}
		isSelfMint = true
		if maxSupply.IsZero() {
			maxSupply = upperLimit
			if limitPerMint.IsZero() {
				limitPerMint = upperLimit
			}
		}
	}
	if maxSupply.IsZero() {
		return
	}

	keyExists, keyRemainingSupply, keyMaxSupply, keyLimitPerMint, keyDecimals, keyInscriptionID, keyIsSelfMint := getTickStatus(tick)

	state.InsertUInt256(keyExists, uint256.NewInt(1))
	state.InsertUInt256(keyRemainingSupply, maxSupply)
	state.InsertUInt256(keyMaxSupply, maxSupply)
	state.InsertUInt256(keyDecimals, decimals)
	state.InsertUInt256(keyLimitPerMint, limitPerMint)

	if isSelfMint {
		state.InsertUInt256(keyIsSelfMint, uint256.NewInt(1))
	} else {
		state.InsertUInt256(keyIsSelfMint, uint256.NewInt(0))
	}

	state.InsertInscriptionID(keyInscriptionID, inscriptionID)
}

func processMint(state KVStorage, mintEvent *getter.BRC20MintEvent) {
	if !mintEvent.Valid {
		return
	}
	upperLimit := getLimit()

	tick := mintEvent.Tick
	keyExists, keyRemainingSupply, _, keyLimitPerMint, keyDecimals, _, _ := getTickStatus(tick)
	tickExists := state.GetUInt256(keyExists)
	if tickExists.Eq(uint256.NewInt(0)) {
		return // not deployed
	}

	decimals := state.GetUInt256(keyDecimals)
	amountString := mintEvent.Amount
	if !isPositiveNumberWithDot(amountString, false) {
		return // invalid amount
	}
	amount, err := get18DecimalNumbers(amountString, decimals, false)
	if err != nil || amount == nil {
		return // invalid amount
	}
	if amount.Gt(upperLimit) || amount.IsZero() {
		return // invalid amount
	}

	remainingSupply := state.GetUInt256(keyRemainingSupply)
	if remainingSupply.IsZero() {
		return // mint ended
	}

	limitPerMint := state.GetUInt256(keyLimitPerMint)
	if limitPerMint != nil && amount.Gt(limitPerMint) {
		return // mint too much
	}
	if amount.Gt(remainingSupply) {
		amount.Set(remainingSupply) // mint remaining token
	}

	f_add := func(v *uint256.Int) *uint256.Int {
		return uint256.NewInt(0).Add(v, amount)
	}

	wallet := mintEvent.ToAddress.Address
	updateBalance(f_add, state, tick, wallet, AvailableBalanceWallet)
	updateBalance(f_add, state, tick, wallet, OverallBalanceWallet)

	f_sub := func(v *uint256.Int) *uint256.Int {
		return uint256.NewInt(0).Sub(v, amount)
	}
	updateTickState(f_sub, state, tick, RemainingSupply)
}

func processInscribeTransfer(state KVStorage, inscribeTransferEvent *getter.BRC20InscribeTransferEvent) {
	if !inscribeTransferEvent.Valid {
		return
	}
	tick := inscribeTransferEvent.Tick
	keyExists, _, _, _, keyDecimals, _, _ := getTickStatus(tick)
	tickExists := state.GetUInt256(keyExists)
	if tickExists.Eq(uint256.NewInt(0)) {
		return // not deployed
	}

	upperLimit := getLimit()
	amountString := inscribeTransferEvent.Amount
	deicmals := state.GetUInt256(keyDecimals)
	if !isPositiveNumberWithDot(amountString, false) {
		return // invalid amount
	}
	amount, err := get18DecimalNumbers(amountString, deicmals, false)
	if err != nil || amount == nil {
		return // invalid amount
	}
	if amount.Gt(upperLimit) || amount.IsZero() {
		return // invalid amount
	}

	wallet := inscribeTransferEvent.ToAddress.Address
	_, _, availableBalance, _ := GetBalances(state, tick, wallet)
	if amount.Gt(availableBalance) {
		return
	}

	f_sub := func(v *uint256.Int) *uint256.Int {
		return uint256.NewInt(0).Sub(v, amount)
	}
	updateBalance(f_sub, state, tick, wallet, AvailableBalanceWallet)

	key := GetEventHash(inscribeTransferEvent.InscriptionID, TransferInscribeCount)
	newEventCount := uint256.NewInt(0).Add(state.GetUInt256(key), uint256.NewInt(1))
	state.InsertUInt256(key, newEventCount)
}

func processTransfer(state KVStorage, transferEvent *getter.BRC20TransferEvent) {
	if !transferEvent.Valid {
		return
	}
	tick := transferEvent.Tick
	keyExists, _, _, _, keyDecimals, _, _ := getTickStatus(tick)
	tickExists := state.GetUInt256(keyExists)
	if tickExists.Eq(uint256.NewInt(0)) {
		return // not deployed
	}

	upperLimit := getLimit()
	amountString := transferEvent.Amount
	decimals := state.GetUInt256(keyDecimals)
	if !isPositiveNumberWithDot(amountString, false) {
		return // invalid amount
	}
	amount, err := get18DecimalNumbers(amountString, decimals, false)
	if err != nil || amount == nil {
		return // invalid amount
	}
	if amount.Gt(upperLimit) || amount.IsZero() {
		return // invalid amount
	}

	fromWallet := transferEvent.FromAddress.Address
	f_sub := func(v *uint256.Int) *uint256.Int {
		return uint256.NewInt(0).Sub(v, amount)
	}
	updateBalance(f_sub, state, tick, fromWallet, OverallBalanceWallet)

	toWallet := transferEvent.ToAddress.Address
	f_add := func(v *uint256.Int) *uint256.Int {
		return uint256.NewInt(0).Add(v, amount)
	}
	updateBalance(f_add, state, tick, toWallet, AvailableBalanceWallet)
	updateBalance(f_add, state, tick, toWallet, OverallBalanceWallet)

	// update transfer-transfer event count
	key := GetEventHash(transferEvent.InscriptionID, TransferTransferCount)
	newEventCount := uint256.NewInt(0).Add(state.GetUInt256(key), uint256.NewInt(1))
	state.InsertUInt256(key, newEventCount)
}

func Exec(state KVStorage, events []getter.BRC20Event, blockHeight uint) {
	if state.GetHeight() != blockHeight-1 {
		panic(fmt.Errorf("mismatched state header: %d and block height: %d", state.GetHeight(), blockHeight-1))
	}
	for _, event := range events {
		switch e := event.(type) {
		case *getter.BRC20DeployEvent:
			processDeploy(state, e)
		case *getter.BRC20MintEvent:
			processMint(state, e)
		case *getter.BRC20TransferEvent:
			processTransfer(state, e)
		case *getter.BRC20InscribeTransferEvent:
			processInscribeTransfer(state, e)
		default:
			panic(fmt.Errorf("unidentified Event Type"))
		}
	}
}
