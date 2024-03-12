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

func GetOrdTransfers(db *gorm.DB, height uint) []OrdTransfer {
	var ordTransfer []OrdTransfer
	sql := `
		SELECT ot.id, ot.inscription_id, ot.old_satpoint, ot.new_pkscript, ot.new_wallet, ot.sent_as_fee, oc."content", oc.content_type
		FROM ord_transfers ot
		LEFT JOIN ord_content oc ON ot.inscription_id = oc.inscription_id
		LEFT JOIN ord_number_to_id onti ON ot.inscription_id = onti.inscription_id
		WHERE ot.block_height = ? 
			AND onti.cursed_for_brc20 = false
			AND oc."content" is not null AND oc."content"->>'p' = 'brc-20'
		ORDER BY ot.id asc;
		`
	db.Raw(sql, height).Scan(&ordTransfer)
	return ordTransfer
}
