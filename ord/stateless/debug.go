package stateless

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"

	"github.com/RiemaLabs/indexer-committee/ord/getter"
	"github.com/RiemaLabs/indexer-committee/ord"
)

type Record struct {
	ID              int
	Pkscript        string
	Wallet          string
	Tick            string
	OverallBalance  string
	AvailableBalance string
	BlockHeight     uint
	EventID         uint
}

func (queue *Queue) DebugRecovery(getter getter.OrdGetter, recoveryTillHeight uint) error {
	curHeight := queue.Header.Height
	startHeight := queue.StartHeight()

	queue.DebugCommitment("Before Recovery")
	// queue.DebugKV("Before Recovery")

	for i := curHeight - 1; i >= recoveryTillHeight-1; i-- {
		// Recover header from i
		index2 := i - startHeight
		pastState := queue.History[index2]
		// pastState := queue.GerDiffAtHeight(i)
		queue.Header.Height = i
		queue.Header.Hash = pastState.Hash

		for _, elem := range pastState.Diff.Elements {
			if elem.OldValueExists {
				queue.Header.KV[elem.Key] = elem.OldValue
				queue.Header.Root.Insert(elem.Key[:], elem.OldValue[:], NodeResolveFn)
			} else {
				queue.Header.Root.Delete(elem.Key[:], NodeResolveFn)
				delete(queue.Header.KV, elem.Key)
			}
		}
		queue.DebugCommitment("Being  Reversed")
		// queue.DebugKV("Being  Reversed")
	}

	log.Print(curHeight, startHeight, recoveryTillHeight)

	for j := recoveryTillHeight - 1; j < curHeight; j++ {
		log.Print("===", j)
		index := j - startHeight
		ordTransfer, err := getter.GetOrdTransfers(j + 1)
		if err != nil {
			return err
		}
		Exec(&queue.Header, ordTransfer)
		var hash string
		hash, err = getter.GetBlockHash(j)
		if err != nil {
			return err
		}
		queue.History[index] = DiffState{
			Height: j,
			Hash:   hash,
			Diff:   queue.Header.Temp,
		}
		queue.Header.Paging(getter, true, NodeResolveFn)
		queue.DebugCommitment("One Step Update")
		// queue.DebugKV("One Step Update")
	}
	return nil
}

func (queue *Queue) DebugUpdate(getter getter.OrdGetter, latestHeight uint) error {
	curHeight := queue.Header.Height
	for i := curHeight + 1; i <= latestHeight; i++ {
		ordTransfer, err := getter.GetOrdTransfers(i)
		if err != nil {
			return err
		}
		Exec(&queue.Header, ordTransfer)
		queue.Offer()
		queue.Header.OrdTrans = ordTransfer
		queue.Header.Paging(getter, true, NodeResolveFn)
	}
	return nil
}

func (queue *Queue) DebugUpdateStrong(getter getter.OrdGetter, latestHeight uint) error {
	// TODO: first load the csv file
	records, err := loadOPIKV("./opiKV/data.csv")
	if err != nil {
		fmt.Println("Error reading file:", err)
		os.Exit(1)
	}

	curHeight := queue.Header.Height
	for i := curHeight + 1; i <= latestHeight; i++ {
		ordTransfer, err := getter.GetOrdTransfers(i)
		if err != nil {
			return err
		}
		Exec(&queue.Header, ordTransfer)
		queue.Offer()
		queue.Header.OrdTrans = ordTransfer
		queue.Header.Paging(getter, true, NodeResolveFn)

		// Start checking
		newHeight := queue.Header.Height
		if newHeight >= 780000 {
			if recordsForHeight, found := records[newHeight]; found {
				for _, ele := range(recordsForHeight) {
					opiTick := ele.Tick
					opiPkScript := ele.Pkscript
					opiOverallBalance := ele.OverallBalance
					opiAvailableBalance := ele.AvailableBalance

					var ordPkScript ord.PkScript = ord.PkScript(opiPkScript)
					_, _, availableBalance, overallBalance := GetBalances(&queue.Header, opiTick, ordPkScript)
					availableBalanceStr := availableBalance.String()
					overallBalanceStr := overallBalance.String()

					if availableBalanceStr != opiAvailableBalance {
						fmt.Printf("Error, not match at %d for availableBalance", newHeight)
					}
					if overallBalanceStr != opiOverallBalance {
						fmt.Printf("Error, not match at %d for overallBalance", newHeight)
					}
				}
			}
		}
	}
	return nil
}

func loadOPIKV(filepath string) (map[uint][]Record, error) {
	file, err := os.Open(filepath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	_, err = reader.Read()
	if err != nil {
		fmt.Println("Error reading first line:", err)
		return nil, err
	}

	records := make(map[uint][]Record)

	for {
		line, err := reader.Read()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			fmt.Println("Error reading line:", err)
			return nil, err
		}

		id, _ := strconv.Atoi(line[0])
		blockHeightUint64, _ := strconv.ParseUint(line[6], 10, 32)
		blockHeight := uint(blockHeightUint64)
		eventID, _ := strconv.ParseUint(line[7], 10, 32)

		// overallBalance, _ := uint256.FromDecimal(line[4])
		// availableBalance, _ := uint256.FromDecimal(line[5])

		record := Record{
			ID:               id,
			Pkscript:         line[1],
			Wallet:           line[2],
			Tick:             line[3],
			OverallBalance:   line[4],
			AvailableBalance: line[5],
			BlockHeight:      uint(blockHeight),
			EventID:          uint(eventID),
		}

		records[blockHeight] = append(records[blockHeight], record)
	}
	return records, nil
}

func (queue *Queue) DebugKV(addition string) {
	filePath := "log2_3.txt"

	KVCommitment := generateMapHash(queue.Header.KV)
	curHeight := queue.Header.Height

	// Use os.Create to create a file for writing
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// Handle the error; you might want to log it or return it
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	// Write the data to the file
	// TODO: write height, addition and commitment into the file in one line, seperate by ====
	data := fmt.Sprintf("%d====%s====%s\n", curHeight, addition, KVCommitment)
	_, err = file.WriteString(data)
	if err != nil {
		// Handle the error
		fmt.Println("Error writing to file:", err)
		return
	}

	// Optionally, report success
	fmt.Println("File written successfully")
}

func (queue *Queue) DebugCommitment(addition string) {
	filePath := "log1_3.txt"

	bytes := queue.Header.Root.Commit().Bytes()
	commitment := base64.StdEncoding.EncodeToString(bytes[:])
	curHeight := queue.Header.Height

	// Use os.Create to create a file for writing
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// Handle the error; you might want to log it or return it
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	// Write the data to the file
	// TODO: write height, addition and commitment into the file in one line, seperate by ====
	data := fmt.Sprintf("%d====%s====%s\n", curHeight, addition, commitment)
	_, err = file.WriteString(data)
	if err != nil {
		// Handle the error
		fmt.Println("Error writing to file:", err)
		return
	}

	// Optionally, report success
	fmt.Println("File written successfully")
}

func generateMapHash(kvMap KeyValueMap) string {
	keys := make([][32]byte, 0, len(kvMap))
	for k := range kvMap {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		return compareByteArrays(keys[i], keys[j])
	})

	var data []byte
	for _, k := range keys {
		data = append(data, k[:]...) // Append key
		temp := kvMap[k]
		data = append(data, temp[:]...) // Correctly append the value
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func compareByteArrays(a, b [32]byte) bool {
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}
