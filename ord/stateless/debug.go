package stateless

import (
	"encoding/csv"
	"log"
	"os"
	"strconv"

	"github.com/RiemaLabs/modular-indexer-committee/ord"
)

type Record struct {
	ID               int
	Pkscript         string
	Wallet           string
	Tick             string
	OverallBalance   string
	AvailableBalance string
	BlockHeight      uint
	EventID          uint
}

type OPIRecords = map[uint][]Record

func LoadOPIRecords(filepath string) (OPIRecords, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	_, err = reader.Read()
	if err != nil {
		return nil, err
	}

	records := make(map[uint][]Record)

	for {
		line, err := reader.Read()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
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

func (h *Header) VerifyState(records *OPIRecords) {
	height := h.Height
	if recordsForHeight, found := (*records)[height]; found {
		for _, ele := range recordsForHeight {
			opiTick := ele.Tick
			opiPkScript := ele.Pkscript
			opiOverallBalance := ele.OverallBalance
			opiAvailableBalance := ele.AvailableBalance

			var ordPkscript ord.Pkscript = ord.Pkscript(opiPkScript)
			_, _, availableBalance, overallBalance := GetBalances(h, opiTick, ordPkscript)
			availableBalanceStr := availableBalance.String()
			overallBalanceStr := overallBalance.String()

			if availableBalanceStr != opiAvailableBalance {
				log.Fatalf(`at block height %d, Pkscript %s's availableBalance doesn't match.
				Our balance is: %s, OPI balance is: %s`, height, ordPkscript, availableBalanceStr, opiAvailableBalance)
			}
			if overallBalanceStr != opiOverallBalance {
				log.Fatalf(`at block height %d, Pkscript %s's availableBalance doesn't match.
				Our balance is: %s, OPI balance is: %s`, height, ordPkscript, availableBalanceStr, opiAvailableBalance)
			}
		}
	}
}
