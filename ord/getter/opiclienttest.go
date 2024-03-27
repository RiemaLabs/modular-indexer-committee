package getter

import (
	"encoding/csv"
	"os"
	"strconv"

	"github.com/RiemaLabs/modular-indexer-committee/ord"
	"gorm.io/gorm"
)

type OPIOrdGetterTest struct {
	db                *gorm.DB
	LatestBlockHeight uint
	BlockHash         map[uint]string
	OrdTransfers      []OrdTransfer
}

func NewOPIOrdGetterTest(config *DatabaseConfig, latestBlockHeight uint) (*OPIOrdGetterTest, error) {
	// Open the CSV file
	file, err := os.Open("./data/782000-ord_transfers.csv")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Parse the CSV file
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	// Initialize OPIOrdGetterTest struct
	getter := OPIOrdGetterTest{
		LatestBlockHeight: latestBlockHeight,
		BlockHash:         make(map[uint]string),
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
	var blockHash string
	sql := `
		SELECT block_hash
		FROM block_hashes
		WHERE block_height = $1
	`
	err := opi.db.Raw(sql, blockHeight).Scan(&blockHash).Error
	if err != nil {
		return "", err
	}
	return blockHash, nil
}

func (opi *OPIOrdGetterTest) GetOrdTransfers(blockHeight uint) ([]OrdTransfer, error) {
	var ordTransfers []OrdTransfer
	sql := `
		SELECT ot.id, ot.inscription_id, ot.block_height, ot.old_satpoint, ot.new_satpoint, ot.new_pkscript, ot.new_wallet, ot.sent_as_fee, oc."content", oc.content_type
		FROM ord_transfers ot
		LEFT JOIN ord_content oc ON ot.inscription_id = oc.inscription_id
		LEFT JOIN ord_number_to_id onti ON ot.inscription_id = onti.inscription_id
		WHERE ot.block_height = $1
			AND onti.cursed_for_brc20 = false
			AND oc."content" is not null AND oc."content"->>'p' = 'brc-20'
		ORDER BY ot.id asc;
		`
	err := opi.db.Raw(sql, blockHeight).Scan(&ordTransfers).Error
	if err != nil {
		return make([]OrdTransfer, 0), err
	}
	return ordTransfers, nil
}
