package ord

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	base58 "github.com/btcsuite/btcd/btcutil/base58"
	bech32 "github.com/btcsuite/btcd/btcutil/bech32"

	uint256 "github.com/holiman/uint256"
)

func convertIntToByte(i *uint256.Int) []byte {
	var dest [32]byte
	i.WriteToArray32(&dest)
	return dest[:]
}

func convertByteToInt(b []byte) *uint256.Int {
	return uint256.NewInt(0).SetBytes(b)
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
