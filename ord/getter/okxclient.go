package getter

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/RiemaLabs/modular-indexer-committee/internal/metrics"
	"github.com/RiemaLabs/modular-indexer-committee/ord"
)

type DatabaseConfig struct {
	Host     string
	User     string
	Password string
	DBname   string
	Port     string
}

type OKXBRC20Getter struct {
	db *gorm.DB
}

func ConnectOKXDatabase(config *DatabaseConfig) (*gorm.DB, error) {
	host := config.Host
	user := config.User
	password := config.Password
	dbname := config.DBname
	port := config.Port
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True", user, password, host, port, dbname)
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}

func NewOKXBRC20Getter(config *DatabaseConfig) (*OKXBRC20Getter, error) {
	db, err := ConnectOKXDatabase(config)
	if err != nil {
		return nil, err
	}
	getter := OKXBRC20Getter{
		db: db,
	}
	return &getter, err
}

func (okx *OKXBRC20Getter) GetLatestBlockHeight() (uint, error) {
	defer metrics.ObserveDBQuery("getLatestBlockHeight", time.Now())

	var blockHeight int
	sql := `
		SELECT latest_block_height
		FROM latest_block_height 
		ORDER BY latest_block_height DESC
		LIMIT 1
	`
	err := okx.db.Raw(sql).Scan(&blockHeight).Error
	if err != nil {
		return 0, err
	}
	return uint(blockHeight), nil
}

func (okx *OKXBRC20Getter) GetBlockHash(blockHeight uint) (string, error) {
	defer metrics.ObserveDBQuery("getBlockHash", time.Now())

	var blockHash string
	sql := `
		SELECT block_hash
		FROM block_hashes
		WHERE block_height = ?
	`
	err := okx.db.Raw(sql, blockHeight).Scan(&blockHash).Error
	if err != nil {
		return "", err
	}
	return blockHash, nil
}

func (okx *OKXBRC20Getter) GetOrdTransfers(blockHeight uint) ([]BRC20Event, error) {
	defer metrics.ObserveDBQuery("getOrdTransfers", time.Now())

	var rows []*row
	result := okx.db.Table("ord_transfers").
		Select("block_height, event_type, tick, inscription_id, inscription_num, old_satpoint, new_satpoint, from_address, to_address, valid, msg, supply, limit_per_mint, decimals, amount").
		Where("block_height = ?", blockHeight).
		Scan(&rows)
	if result.Error != nil {
		return nil, result.Error
	}

	var events []BRC20Event
	for _, row := range rows {
		log.Println(row)
		log.Println(row.InscriptionNum)
		log.Println(row.Decimals)
		intInscriptionNum, err := strconv.ParseInt(row.InscriptionNum, 10, 32)
		if err != nil {
			return nil, err
		}
		baseEvent := BaseEvent{
			BlockHeight:    row.BlockHeight,
			EventType:      row.EventType,
			Tick:           row.Tick,
			InscriptionID:  row.InscriptionID,
			InscriptionNum: int32(intInscriptionNum),
			OldSatpoint:    row.OldSatpoint,
			NewSatpoint:    row.NewSatpoint,
			FromAddress:    Address{Address: ord.Wallet(row.FromAddress)},
			ToAddress:      Address{Address: ord.Wallet(row.ToAddress)},
			Valid:          row.Valid,
			Msg:            row.Msg,
		}

		switch row.EventType {
		case "deploy":
			intDecimal, err := strconv.ParseInt(row.Decimals, 10, 32)
			if err != nil {
				return nil, err
			}
			events = append(events, &BRC20DeployEvent{
				BaseEvent:    baseEvent,
				Supply:       row.Supply,
				LimitPerMint: row.LimitPerMint,
				Decimal:      int32(intDecimal),
			})
		case "mint":
			events = append(events, &BRC20MintEvent{
				BaseEvent: baseEvent,
				Amount:    row.Amount,
			})
		case "transfer":
			events = append(events, &BRC20TransferEvent{
				BaseEvent: baseEvent,
				Amount:    row.Amount,
			})
		case "inscribeTransfer":
			events = append(events, &BRC20InscribeTransferEvent{
				BaseEvent: baseEvent,
				Amount:    row.Amount,
			})
		default:
			fmt.Println("Unknown event type: ", row.EventType)
		}
	}

	return events, nil
}
