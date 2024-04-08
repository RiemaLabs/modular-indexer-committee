package stateless

import (
	"fmt"
	"strings"
	"unicode"

	base58 "github.com/btcsuite/btcd/btcutil/base58"
	"github.com/ethereum/go-verkle"

	uint256 "github.com/holiman/uint256"
)

// The first block height of the brc-20 protocol.
const BRC20StartHeight uint = 779832

var NodeResolveFn verkle.NodeResolverFn = nil

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

	// Create a uint256.Int representation of (10^18)
	ten18 := uint256.NewInt(0)
	for i := 0; i < 18; i++ {
		ten18 = ten18.Mul(ten18, uint256.NewInt(10))
		if i == 0 { // Initialize to 10 on the first iteration
			ten18 = uint256.NewInt(10)
		}
	}

	// Calculate (2^64 - 1) * (10^18)
	result := uint256.NewInt(0).Mul(two64Minus1, ten18)
	return result
}

func decodeBitcoinWallet(s string) []byte {
	return base58.Decode(s)
}

func encodeBitcoinWallet(b []byte) string {
	return base58.Encode(b)
}

func defaultValue() [ValueSize]byte {
	// TODO: Medium. Optimize style.
	return [ValueSize]byte{
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
	}
}

func bytesTo32Bytes(b []byte) [32]byte {
	var newArray [32]byte
	copy(newArray[:], b[:])
	return newArray
}
