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

type ORDRecords = map[uint][]Record

func LoadORDRecords(filepath string) (ORDRecords, error) {
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

func (h *Header) VerifyState(records *ORDRecords) {
	height := h.Height
	if recordsForHeight, found := (*records)[height]; found {
		for _, ele := range recordsForHeight {
			ordTick := ele.Tick
			ordWallet := ele.Wallet
			ordOverallBalance := ele.OverallBalance
			ordAvailableBalance := ele.AvailableBalance

			_, _, availableBalance, overallBalance := GetBalances(h, ordTick, ord.Wallet(ordWallet))
			availableBalanceStr := availableBalance.String()
			overallBalanceStr := overallBalance.String()

			if availableBalanceStr != ordAvailableBalance {
				log.Fatalf(`at block height %d, Wallet %s's availableBalance doens't match.
				Our balance is: %s, ORD balance is: %s`, height, ordWallet, availableBalanceStr, ordAvailableBalance)
			}
			if overallBalanceStr != ordOverallBalance {
				log.Fatalf(`at block height %d, Wallet %s's availableBalance doens't match.
				Our balance is: %s, ORD balance is: %s`, height, ordWallet, availableBalanceStr, ordAvailableBalance)
			}
		}
	}
}
