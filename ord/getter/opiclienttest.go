package getter

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	"github.com/RiemaLabs/modular-indexer-committee/ord"
)

type OPIOrdGetterTest struct {
	LatestBlockHeight uint
	BlockHash         map[uint]string
	OrdTransfers      []OrdTransfer
}

func NewOPIOrdGetterTest(config *DatabaseConfig, latestBlockHeight uint) (*OPIOrdGetterTest, error) {
	// Initialize OPIOrdGetterTest struct
	getter := OPIOrdGetterTest{
		LatestBlockHeight: latestBlockHeight,
		BlockHash:         make(map[uint]string),
	}

	// read data/*-brc20_block_hashes.csv and populate the BlockHash map in the OPIOrdGetterTest struct
	file, err := os.Open("./data/782000-brc20_block_hashes.csv")
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

	// read data/*-ord_transfers.csv and populate the OrdTransfers slice in the OPIOrdGetterTest struct
	file, err = os.Open("./data/782000-ord_transfers.csv")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader = csv.NewReader(file)
	records, err = reader.ReadAll()
	if err != nil {
		return nil, err
	}

	// Parse CSV records into OrdTransfer structs
	// "id","inscription_id","block_height","old_satpoint","new_satpoint","new_pkscript","new_wallet","sent_as_fee","content","content_type"
	for _, record := range records[1:] { // Skip header row
		id, _ := strconv.Atoi(record[0])
		inscriptionID := record[1]
		blockHeight, _ := strconv.Atoi(record[2])
		oldSatpoint := record[3]
		newSatpoint := record[4]
		newPkscript := ord.Pkscript(record[5])
		new_wallet := ord.Wallet(record[6])
		sent_as_fee, _ := strconv.ParseBool(record[7])
		content := record[8]
		content_type := record[9]

		// Create OrdTransfer struct
		ordTransfer := OrdTransfer{
			ID:            uint(id),
			InscriptionID: inscriptionID,
			BlockHeight:   uint(blockHeight),
			OldSatpoint:   oldSatpoint,
			NewSatpoint:   newSatpoint,
			NewPkscript:   newPkscript,
			NewWallet:     new_wallet,
			SentAsFee:     bool(sent_as_fee),
			Content:       []byte(content),
			ContentType:   content_type,
		}

		// Append to OrdTransfers slice
		getter.OrdTransfers = append(getter.OrdTransfers, ordTransfer)
	}

	return &getter, nil
}

func (opi *OPIOrdGetterTest) GetLatestBlockHeight() (uint, error) {
	return opi.LatestBlockHeight, nil
}

func (opi *OPIOrdGetterTest) GetBlockHash(blockHeight uint) (string, error) {
	if result, found := opi.BlockHash[blockHeight]; found {
		return result, nil
	}
	return "", fmt.Errorf("block hash not found for block height: %d", blockHeight)
}

func (opi *OPIOrdGetterTest) GetOrdTransfers(blockHeight uint) ([]OrdTransfer, error) {
	var filteredOrdTransfers []OrdTransfer
	for _, transfer := range opi.OrdTransfers {
		// Filter by block height by iterating over OrdTransfers slice
		if transfer.BlockHeight == blockHeight {
			filteredOrdTransfers = append(filteredOrdTransfers, transfer)
		}
	}
	return filteredOrdTransfers, nil
}
