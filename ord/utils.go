package ord

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// The number of confirmations to be considered immutable and can't be re-organized.
const BitcoinConfirmations uint = 6

type OrdTransfer struct {
	ID            uint
	InscriptionID string
	BlockHeight   uint
	OldSatpoint   string
	NewSatpoint   string
	NewPkscript   string
	NewWallet     string
	SentAsFee     bool
	Content       []byte
	ContentType   string
	ParentID      string
}

type BRC20Hashes struct {
	ID          uint
	BlockHeight uint
	BlockHash   string
}

func ConnectDatabase() *gorm.DB {
	dsn := "host=127.0.0.1 user=postgres password=170501 dbname=postgres port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	return db
}

func GenerateOrdTransfers(height uint) {
	db := ConnectDatabase()
	filename := "./data/" + fmt.Sprintf("%d", height) + "-ord_transfers.csv"
	csvFile, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()

	data := [][]string{
		{"id", "inscription_id", "block_height", "old_satpoint", "new_satpoint", "new_pkscript", "new_wallet", "sent_as_fee", "content", "content_type", "parent_id"},
	}
	writer := csv.NewWriter(csvFile)
	defer writer.Flush()
	for _, record := range data {
		if err := writer.Write(record); err != nil {
			panic(err)
		}
	}

	sql := `
			SELECT ot.id, ot.inscription_id, ot.block_height, ot.old_satpoint, ot.new_satpoint, ot.new_pkscript, ot.new_wallet, ot.sent_as_fee, oc."content", oc.content_type, onti.parent_id
			FROM ord_transfers ot
			LEFT JOIN ord_content oc ON ot.inscription_id = oc.inscription_id
			LEFT JOIN ord_number_to_id onti ON ot.inscription_id = onti.inscription_id
			WHERE ot.block_height = ?
				AND onti.cursed_for_brc20 = false
				AND oc."content" is not null AND oc."content"->>'p' = 'brc-20'
			ORDER BY ot.id asc;
			`

	for i := 779832; i <= int(height); i++ {
		var ordTransfers []OrdTransfer

		db.Raw(sql, i).Scan(&ordTransfers)

		if i%100 == 0 {
			log.Printf("Height: %d, Total: %d\n", i, len(ordTransfers))
		}

		data = [][]string{}

		for _, d := range ordTransfers {
			dStr := []string{
				strconv.FormatUint(uint64(d.ID), 10),
				d.InscriptionID,
				strconv.FormatUint(uint64(d.BlockHeight), 10),
				d.OldSatpoint,
				d.NewSatpoint,
				d.NewPkscript,
				d.NewWallet,
				strconv.FormatBool(d.SentAsFee),
				string(d.Content),
				d.ContentType,
				d.ParentID,
			}
			data = append(data, dStr)
		}

		for _, record := range data {
			if err := writer.Write(record); err != nil {
				panic(err)
			}
		}
	}

	log.Println("Data written into file!")
}

func GenerateBRC20BlockHashes(height uint) {
	db := ConnectDatabase()
	filename := "./data/" + fmt.Sprintf("%d", height) + "-brc20_block_hashes.csv"
	csvFile, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()

	data := [][]string{
		{"block_height", "block_hash"},
	}
	writer := csv.NewWriter(csvFile)
	defer writer.Flush()
	for _, record := range data {
		if err := writer.Write(record); err != nil {
			panic(err)
		}
	}

	sql := `SELECT block_height, block_hash FROM public.brc20_block_hashes
	WHERE block_height >= 779832 AND block_height <= ?
	ORDER BY id ASC;`

	var brc20Hashes []BRC20Hashes

	db.Raw(sql, height).Scan(&brc20Hashes)

	data = [][]string{}

	for _, d := range brc20Hashes {
		dStr := []string{
			strconv.FormatUint(uint64(d.BlockHeight), 10),
			d.BlockHash,
		}
		data = append(data, dStr)
	}

	for _, record := range data {
		if err := writer.Write(record); err != nil {
			panic(err)
		}
	}

	log.Println("Data written into file!")
}
