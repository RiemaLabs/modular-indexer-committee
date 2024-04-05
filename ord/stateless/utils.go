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

// func generateRandomPoints(numPoints uint64) []banderwagon.Element {
// 	seed := "eth_verkle_oct_2021"

// 	points := []banderwagon.Element{}

// 	var increment uint64 = 0

// 	for uint64(len(points)) != numPoints {

// 		digest := sha256.New()
// 		digest.Write([]byte(seed))

// 		b := make([]byte, 8)
// 		binary.BigEndian.PutUint64(b, increment)
// 		digest.Write(b)

// 		hash := digest.Sum(nil)

// 		var x fp.Element
// 		x.SetBytes(hash)

// 		increment++

// 		x_as_bytes := x.Bytes()
// 		var point_found banderwagon.Element
// 		err := point_found.SetBytes(x_as_bytes[:])
// 		if err != nil {
// 			// This point is not in the correct subgroup or on the curve
// 			continue
// 		}
// 		points = append(points, point_found)

// 	}

// 	return points
// }

// func computeNumRounds(vectorSize uint32) uint32 {
// 	// Check if this number is 0
// 	// zero is not a valid input to this function for our usecase
// 	if vectorSize == 0 {
// 		panic("zero is not a valid input")
// 	}

// 	// See: https://stackoverflow.com/a/600306
// 	isPow2 := (vectorSize & (vectorSize - 1)) == 0

// 	if !isPow2 {
// 		panic("non power of 2 numbers are not valid inputs")
// 	}

// 	res := math.Log2(float64(vectorSize))

// 	return uint32(res)
// }

// func getNewIPASettings() (*ipa.IPAConfig, error) {
// 	srs := generateRandomPoints(common.VectorLength)
// 	precompMSM, err := banderwagon.NewPrecompMSM(srs)
// 	if err != nil {
// 		return nil, fmt.Errorf("creating precomputed MSM: %s", err)
// 	}
// 	return &ipa.IPAConfig{
// 		SRS:                srs,
// 		Q:                  banderwagon.Generator,
// 		PrecompMSM:         precompMSM,
// 		PrecomputedWeights: ipa.NewPrecomputedWeights(),
// 		numRounds:          computeNumRounds(common.VectorLength),
// 	}, nil
// }
