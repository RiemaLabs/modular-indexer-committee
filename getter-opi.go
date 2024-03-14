package main

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectOPIDatabase(config Config) (*gorm.DB, error) {
	host := GlobalConfig.Database.Host
	user := GlobalConfig.Database.User
	password := GlobalConfig.Database.Password
	dbname := GlobalConfig.Database.DBname
	port := GlobalConfig.Database.Port
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s", host, user, password, dbname, port)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

type OPIBitcoinGetter struct {
	db *gorm.DB
}

func NewOPIBitcoinGetter(config Config) (*OPIBitcoinGetter, error) {
	db, err := ConnectOPIDatabase(config)
	if err != nil {
		return nil, err
	}
	getter := OPIBitcoinGetter{
		db: db,
	}
	return &getter, err
}

func (opi *OPIBitcoinGetter) GetLatestBlockHeight() (uint, error) {
	var blockHeight int
	sql := `
		SELECT block_height
		FROM block_hashes ORDER BY block_height DESC LIMIT 1
	`
	err := opi.db.Raw(sql).Scan(&blockHeight).Error
	if err != nil {
		return 0, err
	}
	return uint(blockHeight), nil
}

func (opi *OPIBitcoinGetter) GetBlockHash(blockHeight uint) (string, error) {
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

func (opi *OPIBitcoinGetter) GetOrdTransfers(blockHeight uint) ([]OrdTransfer, error) {
	var ordTransfer []OrdTransfer
	sql := `
		SELECT ot.id, ot.inscription_id, ot.old_satpoint, ot.new_pkscript, ot.new_wallet, ot.sent_as_fee, oc."content", oc.content_type
		FROM ord_transfers ot
		LEFT JOIN ord_content oc ON ot.inscription_id = oc.inscription_id
		LEFT JOIN ord_number_to_id onti ON ot.inscription_id = onti.inscription_id
		WHERE ot.block_height = $1
			AND onti.cursed_for_brc20 = false
			AND oc."content" is not null AND oc."content"->>'p' = 'brc-20'
		ORDER BY ot.id asc;
		`
	err := opi.db.Raw(sql, blockHeight).Scan(&ordTransfer).Error
	if err != nil {
		return make([]OrdTransfer, 0), err
	}
	return ordTransfer, nil
}
