package getter

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/RiemaLabs/modular-indexer-committee/internal/metrics"
)

type DatabaseConfig struct {
	Host     string
	User     string
	Password string
	DBname   string
	Port     string
}

type OPIOrdGetter struct {
	db *gorm.DB
}

func ConnectOPIDatabase(config *DatabaseConfig) (*gorm.DB, error) {
	host := config.Host
	user := config.User
	password := config.Password
	dbname := config.DBname
	port := config.Port
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s", host, user, password, dbname, port)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

func NewOPIOrdGetter(config *DatabaseConfig) (*OPIOrdGetter, error) {
	db, err := ConnectOPIDatabase(config)
	if err != nil {
		return nil, err
	}
	getter := OPIOrdGetter{
		db: db,
	}
	return &getter, err
}

func (opi *OPIOrdGetter) GetLatestBlockHeight() (uint, error) {
	defer metrics.ObserveDBQuery("getLatestBlockHeight", time.Now())

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

func (opi *OPIOrdGetter) GetBlockHash(blockHeight uint) (string, error) {
	defer metrics.ObserveDBQuery("getBlockHash", time.Now())

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

func (opi *OPIOrdGetter) GetOrdTransfers(blockHeight uint) ([]OrdTransfer, error) {
	defer metrics.ObserveDBQuery("getOrdTransfers", time.Now())

	var ordTransfers []OrdTransfer
	sql := `
	SELECT ot.id, ot.inscription_id, ot.block_height, ot.old_satpoint, ot.new_satpoint, ot.new_pkscript, ot.new_wallet, ot.sent_as_fee, oc."content", oc.content_type, onti.parent_id
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
