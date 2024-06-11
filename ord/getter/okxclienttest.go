package getter

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	"github.com/RiemaLabs/modular-indexer-committee/ord"
)

type OKXBRC20GetterTest struct {
	latestBlockHeight uint
	BlockHash         map[uint]string
	OrdTransfers      []BRC20Event
}

func NewOKXBRC20GetterTest(config *DatabaseConfig, latestBlockHeight uint, hashedHeight uint) (*OKXBRC20GetterTest, error) {
	// Initialize OKXBRC20GetterTest struct
	getter := OKXBRC20GetterTest{
		latestBlockHeight: latestBlockHeight,
		BlockHash:         make(map[uint]string),
	}

	// read data/*-brc20_block_hashes.csv and populate the BlockHash map in the OKXBRC20GetterTest struct
	filename := "./data/" + fmt.Sprintf("%d", hashedHeight) + "-okx-brc20_block_hashes.csv"
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	for _, record := range records[1:] { // Skip header row
		blockHeight, _ := strconv.Atoi(record[0])
		blockHash := record[1]
		getter.BlockHash[uint(blockHeight)] = blockHash
	}

	// read data/*-ord_transfers.csv and populate the OrdTransfers slice in the OKXBRC20GetterTest struct
	filename = "./data/" + fmt.Sprintf("%d", hashedHeight) + "-okx-ord_transfers.csv"
	file, err = os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader = csv.NewReader(file)
	records, err = reader.ReadAll()
	if err != nil {
		return nil, err
	}

	// Parse CSV records into BRC20 event structs

	for _, record := range records[1:] { // Skip header row
		blockHeight, _ := strconv.Atoi(record[0])
		eventType := record[1]
		tick := record[2]
		inscriptionID := record[3]
		inscriptionNum, _ := strconv.Atoi(record[4])
		oldSatpoint := record[5]
		newSatpoint := record[6]
		fromAddress := ord.Wallet(record[7])
		toAddress := ord.Wallet(record[8])
		valid, _ := strconv.ParseBool(record[9])
		msg := record[10]

		// Append to OrdTransfers slice
		baseEvent := BaseEvent{
			BlockHeight:    uint(blockHeight),
			EventType:      eventType,
			Tick:           tick,
			InscriptionID:  inscriptionID,
			InscriptionNum: int32(inscriptionNum),
			OldSatpoint:    oldSatpoint,
			NewSatpoint:    newSatpoint,
			FromAddress:    Address{Address: fromAddress},
			ToAddress:      Address{Address: toAddress},
			Valid:          valid,
			Msg:            msg,
		}

		switch eventType {
		case "deploy":
			supply := record[11]
			limitPerMint := record[12]
			decimal, _ := strconv.Atoi(record[13])
			deployEvent := BRC20DeployEvent{
				BaseEvent:    baseEvent,
				Supply:       supply,
				LimitPerMint: limitPerMint,
				Decimal:      int32(decimal),
			}
			getter.OrdTransfers = append(getter.OrdTransfers, &deployEvent)
		case "mint":
			amount := record[14]
			mintEvent := BRC20MintEvent{
				BaseEvent: baseEvent,
				Amount:    amount,
			}
			getter.OrdTransfers = append(getter.OrdTransfers, &mintEvent)
		case "transfer":
			amount := record[14]
			transferEvent := BRC20TransferEvent{
				BaseEvent: baseEvent,
				Amount:    amount,
			}
			getter.OrdTransfers = append(getter.OrdTransfers, &transferEvent)
		case "inscribeTransfer":
			amount := record[14]
			inscribeTransferEvent := BRC20InscribeTransferEvent{
				BaseEvent: baseEvent,
				Amount:    amount,
			}
			getter.OrdTransfers = append(getter.OrdTransfers, &inscribeTransferEvent)
		default:
			fmt.Printf("Unknown event type: %s\n", eventType)
		}
	}

	return &getter, nil
}

func (okx *OKXBRC20GetterTest) GetLatestBlockHeight() (uint, error) {
	return okx.latestBlockHeight, nil
}

func (okx *OKXBRC20GetterTest) SetLatestBlockHeight(height uint) {
	okx.latestBlockHeight = height
}

func (okx *OKXBRC20GetterTest) GetBlockHash(blockHeight uint) (string, error) {
	if result, found := okx.BlockHash[blockHeight]; found {
		return result, nil
	}
	return "", fmt.Errorf("block hash not found for block height: %d", blockHeight)
}

func (okx *OKXBRC20GetterTest) GetOrdTransfers(blockHeight uint) ([]BRC20Event, error) {
	var filteredOrdTransfers []BRC20Event
	for _, event := range okx.OrdTransfers {
		switch e := event.(type) {
		case *BRC20DeployEvent:
			if e.BlockHeight == blockHeight {
				filteredOrdTransfers = append(filteredOrdTransfers, event)
			}
		case *BRC20MintEvent:
			if e.BlockHeight == blockHeight {
				filteredOrdTransfers = append(filteredOrdTransfers, event)
			}
		case *BRC20TransferEvent:
			if e.BlockHeight == blockHeight {
				filteredOrdTransfers = append(filteredOrdTransfers, event)
			}
		case *BRC20InscribeTransferEvent:
			if e.BlockHeight == blockHeight {
				filteredOrdTransfers = append(filteredOrdTransfers, event)
			}
		}
	}
	return filteredOrdTransfers, nil
}
