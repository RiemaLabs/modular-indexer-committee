package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode"

	base58 "github.com/btcsuite/btcd/btcutil/base58"
	bech32 "github.com/btcsuite/btcd/btcutil/bech32"

	verkle "github.com/ethereum/go-verkle"
	uint256 "github.com/holiman/uint256"
	"golang.org/x/crypto/sha3"
)

var nodeResolveFn verkle.NodeResolverFn = nil

func convertIntToByte(i *uint256.Int) []byte {
	var dest [32]byte
	i.WriteToArray32(&dest)
	return dest[:]
}

func convertByteToInt(b []byte) *uint256.Int {
	return uint256.NewInt(0).SetBytes(b)
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

func isPositiveNumber(s string, doStrip bool) bool {
	if doStrip {
		s = strings.TrimSpace(s)
	}
	if len(s) == 0 {
		return false
	}
	for _, ch := range s {
		if !unicode.IsDigit(ch) {
			return false
		}
	}
	return true
}

func isPositiveNumberWithDot(s string, doStrip bool) bool {
	if doStrip {
		s = strings.TrimSpace(s)
	}
	if len(s) == 0 || s[0] == '.' || s[len(s)-1] == '.' {
		return false
	}
	dotFound := false
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			if ch != '.' || dotFound {
				return false
			}
			dotFound = true
		}
	}
	return true
}

func getNumberExtendedTo18Decimals(s string, decimals *uint256.Int, doStrip bool) (*uint256.Int, error) {
	if doStrip {
		s = strings.TrimSpace(s)
	}

	eighteen := uint256.NewInt(18)

	if strings.Contains(s, ".") {
		parts := strings.Split(s, ".")
		normalPart := parts[0]
		decimalPart := parts[1]

		decimalLength := uint256.NewInt(uint64(len(decimalPart)))

		if decimalLength.Gt(decimals) || len(decimalPart) == 0 {
			// More decimal digits than allowed or no decimal digits
			return nil, nil
		}

		// Ensure decimal part is not longer than decimals and extend to 18 digits
		requiredZeros := eighteen.Sub(eighteen, decimalLength)
		decimalPart += strings.Repeat("0", int(requiredZeros.Uint64()))

		// Convert the concatenated string to *uint256.Int
		result, err := uint256.FromDecimal(normalPart + decimalPart)
		if err != nil {
			return nil, fmt.Errorf("number overflow: %s", normalPart+decimalPart)
		}
		return result, nil
	} else {
		// No decimal point, directly extend to 18 digits
		result, err := uint256.FromDecimal(s + strings.Repeat("0", 18))
		if err != nil {
			return nil, fmt.Errorf("number overflow: %s", s)
		}
		return result, nil
	}
}

func getLimit() *uint256.Int {
	two64Minus1 := uint256.NewInt(0).Sub(uint256.NewInt(0).Lsh(uint256.NewInt(1), 64), uint256.NewInt(1))

	// 创建(10^18)的uint256.Int表示
	ten18 := uint256.NewInt(0)
	for i := 0; i < 18; i++ {
		ten18 = ten18.Mul(ten18, uint256.NewInt(10))
		if i == 0 { // 初始化为10在第一次迭代
			ten18 = uint256.NewInt(10)
		}
	}

	// 计算(2^64 - 1) * (10^18)
	result := uint256.NewInt(0).Mul(two64Minus1, ten18)
	return result
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

func getValueOrZero(stateRoot verkle.VerkleNode, key []byte) *uint256.Int {
	res := uint256.NewInt(0)
	value, _ := stateRoot.Get(key, nodeResolveFn)
	if len(value) == 0 {
		return res
	}
	return res.SetBytes(value)
}

// save decoded wallet address and pkscript
func saveSourceWalletAndPkscript(stateRoot verkle.VerkleNode, inscrId string, sourceAddr string, pkScript string) {
	eventKey := getEventHash("transfer-inscribe-source-wallet", inscrId)
	stateRoot.Insert(eventKey, []byte(sourceAddr), nodeResolveFn)

	length := len(pkScript)
	prefix := []byte{byte(length)}
	if len(pkScript)%2 == 1 {
		pkScript += "0"
	}
	encodedPkscript, _ := hex.DecodeString(pkScript)
	encodedPkscript = append(prefix, encodedPkscript...)
	pkScriptKey1 := getEventHash("transfer-inscribe-source-pkscript-1", inscrId)
	b1, _ := padTo32Bytes(encodedPkscript[:min(len(encodedPkscript), 32)])
	stateRoot.Insert(pkScriptKey1, b1, nodeResolveFn)
	if len(encodedPkscript) > 32 {
		pkScriptKey2 := getEventHash("transfer-inscribe-source-pkscript-2", inscrId)
		b2, _ := padTo32Bytes(encodedPkscript[32:])
		stateRoot.Insert(pkScriptKey2, b2, nodeResolveFn)
	}
}

// get decoded wallet address and pkscript
func getSourceWalletAndPkscript(stateRoot verkle.VerkleNode, inscrId string) (string, string) {
	eventKey := getEventHash("transfer-inscribe-source-wallet", inscrId)
	sourceAddr, _ := stateRoot.Get(eventKey, nodeResolveFn)

	pkScriptKey1, pkScriptKey2 := getEventHash("transfer-inscribe-source-pkscript-1", inscrId), getEventHash("transfer-inscribe-source-pkscript-2", inscrId)
	b1, _ := stateRoot.Get(pkScriptKey1, nodeResolveFn)
	b2, _ := stateRoot.Get(pkScriptKey2, nodeResolveFn)
	b := append(b1, b2...)
	length := int(b[0])
	sourcePkscript := hex.EncodeToString(b[1:])[:length]
	return string(sourceAddr), sourcePkscript
}
