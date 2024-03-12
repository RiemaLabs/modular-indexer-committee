package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"strings"
	"unicode"

	verkle "github.com/ethereum/go-verkle"
	uint256 "github.com/holiman/uint256"
	"gorm.io/gorm"
)

var debug_dict = make(map[string]string)

func getMACAddress() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Fatal(err)
	}

	for _, iface := range interfaces {
		// Skip down interface
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		// Skip loopback interface
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		// Get the MAC address
		mac := iface.HardwareAddr
		if len(mac) == 0 {
			continue
		}
		// Return the first MAC address found
		return mac.String()
	}

	// Return empty string if no MAC address was found
	return ""
}

func uploadFile(filePath, targetURL string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// create a buffer for multipart writting器
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// create the table
	part, err := writer.CreateFormFile("file", filePath)
	if err != nil {
		return err
	}

	// read from the file
	_, err = io.Copy(part, file)
	if err != nil {
		return err
	}

	// close the writer
	err = writer.Close()
	if err != nil {
		return err
	}

	// create request
	request, err := http.NewRequest("POST", targetURL, body)
	if err != nil {
		return err
	}

	// set up the header and content type
	request.Header.Set("Content-Type", writer.FormDataContentType())

	// send request
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// check if sending is successful
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Server False: %d", resp.StatusCode)
	}

	return nil
}

func getMaxBlockHeight() (uint, error) {
	// API URL to get the latest block height
	// resp, err := http.Get("https://blockchain.info/latestblock")
	// if err != nil {
	// 	return 0, err
	// }
	// defer resp.Body.Close()

	// body, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	return 0, err
	// }

	// var result struct {
	// 	Height int `json:"height"`
	// }

	// if err = json.Unmarshal(body, &result); err != nil {
	// 	return 0, err
	// }

	// return uint(result.Height), nil
	return 831947, nil
}

func getBlockHash(blockHeight uint) (string, error) {
	// API URL to get the block hash
	resp, err := http.Get(fmt.Sprintf("https://blockchain.info/block-height/%d?format=json", blockHeight))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		Blocks []struct {
			Hash string `json:"hash"`
		} `json:"blocks"`
	}

	if err = json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.Blocks) > 0 {
		return result.Blocks[0].Hash, nil
	}

	return "", fmt.Errorf("no blocks found at height %d", blockHeight)
}

func convertByteToInt(b []byte) *uint256.Int {
	return uint256.NewInt(0).SetBytes(b)
}

func convertStringTo32Byte(s string) []byte {
	var b [32]byte
	copy(b[:], s)
	return b[:]
}

func convert32ByteToString(b []byte) string {
	return string(b)
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

func getStateDiff(db *gorm.DB, blockHeight uint) map[string][]byte {
	var diffBalances []BRC20HistoricBalances
	sql := `
		SELECT * FROM public.brc20_historic_balances where block_height = ?
		ORDER BY id ASC
		`
	db.Raw(sql, blockHeight).Scan(&diffBalances)

	diffState := make(map[string][]byte)
	for _, diff := range diffBalances {
		availableBalance := uint256.MustFromDecimal(diff.AvailableBalance)
		diffState[string(getHash("available-balance", diff.Tick, diff.Pkscript))] = convertIntToByte(availableBalance)
		debug_dict[string(getHash("available-balance", diff.Tick, diff.Pkscript))] = diff.Tick + ", pkscript: " + diff.Pkscript + ", available-balance"
		walletAddrByte, _ := decodeBitcoinAddress(diff.Wallet)
		walletAddr := string(walletAddrByte)
		diffState[string(getHash("available-balance", diff.Tick, walletAddr))] = convertIntToByte(availableBalance)
		debug_dict[string(getHash("available-balance", diff.Tick, walletAddr))] = diff.Tick + ", wallet: " + diff.Wallet + ", available-balance"

		overallBalance := uint256.MustFromDecimal(diff.OverallBalance)
		diffState[string(getHash("overall-balance", diff.Tick, diff.Pkscript))] = convertIntToByte(overallBalance)
		debug_dict[string(getHash("overall-balance", diff.Tick, diff.Pkscript))] = diff.Tick + ", pkscript: " + diff.Pkscript + ", overall-balance"
		diffState[string(getHash("overall-balance", diff.Tick, walletAddr))] = convertIntToByte(overallBalance)
		debug_dict[string(getHash("overall-balance", diff.Tick, walletAddr))] = diff.Tick + ", wallet: " + diff.Wallet + ", overall-balance"
	}
	return diffState
}

func getGlobalState(db *gorm.DB, blockHeight uint) []BRC20StateDiff {
	var diffBalances []BRC20HistoricBalances
	db.Where("block_height <= ?", blockHeight).Unscoped().Find(&diffBalances)
	var diffState []BRC20StateDiff
	for _, diff := range diffBalances {
		availableBalance := uint256.MustFromDecimal(diff.AvailableBalance)
		diffState = append(diffState, BRC20StateDiff{
			Key:   string(getHash("available-balance", diff.Tick, diff.Pkscript)),
			Value: convertIntToByte(availableBalance),
		})
	}
	return diffState
}

func getDeployedTicksAtHeight(db *gorm.DB, blockHeight uint) map[string][]byte {
	var deployedTicks []BRC20Tickers
	db.Where("block_height=?", blockHeight).Unscoped().Find(&deployedTicks)
	diffState := make(map[string][]byte)
	for _, deployedTick := range deployedTicks {
		tick, _, limitPerMintString, decimalsString := deployedTick.Tick, deployedTick.RemainingSupply, deployedTick.LimitPerMint, deployedTick.Decimals
		keyTick, _, keyLPM, keyD := getTickHash(tick)

		limitPerMint := uint256.MustFromDecimal(limitPerMintString)
		decimals := uint256.MustFromDecimal(decimalsString)
		diffState[string(keyTick)] = convertIntToByte(uint256.NewInt(0))
		diffState[string(keyLPM)] = convertIntToByte(limitPerMint)
		diffState[string(keyD)] = convertIntToByte(decimals)

		debug_dict[string(keyTick)] = tick + ", existence"
		debug_dict[string(keyLPM)] = tick + ", limit per mint"
		debug_dict[string(keyD)] = tick + ", decimals"
	}
	return diffState
}

func getDeployedTicks(db *gorm.DB, blockHeight uint) []BRC20StateDiff {
	var deployedTicks []BRC20Tickers
	db.Where("block_height<?", blockHeight).Unscoped().Find(&deployedTicks)
	var diffState []BRC20StateDiff
	for _, deployedTick := range deployedTicks {
		tick, remainingSupplyString, limitPerMintString, decimalsString := deployedTick.Tick, deployedTick.RemainingSupply, deployedTick.LimitPerMint, deployedTick.Decimals
		keyTick, keyRS, keyLPM, keyD := getTickHash(tick)
		remainingSupply, err := uint256.FromDecimal(remainingSupplyString)
		if err != nil {
			continue
		}
		limitPerMint, err := uint256.FromDecimal(limitPerMintString)
		if err != nil {
			continue
		}
		decimals, err := uint256.FromDecimal(decimalsString)
		if err != nil {
			continue
		}
		diffState = append(diffState, BRC20StateDiff{
			Key:   string(keyTick),
			Value: convertIntToByte(uint256.NewInt(0)),
		})
		diffState = append(diffState, BRC20StateDiff{
			Key:   string(keyRS),
			Value: convertIntToByte(remainingSupply),
		})
		diffState = append(diffState, BRC20StateDiff{
			Key:   string(keyLPM),
			Value: convertIntToByte(limitPerMint),
		})
		diffState = append(diffState, BRC20StateDiff{
			Key:   string(keyD),
			Value: convertIntToByte(decimals),
		})
	}
	return diffState
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
