package getter

import (
	"gorm.io/gorm"
)

type OPIOrdGetterTest struct {
	db                *gorm.DB
	LatestBlockHeight uint
	BlockHash         map[uint]string
}

func NewOPIOrdGetterTest(config *DatabaseConfig) (*OPIOrdGetterTest, error) {
	db, err := ConnectOPIDatabase(config)
	if err != nil {
		return nil, err
	}
	getter := OPIOrdGetterTest{
		db:                db,
		LatestBlockHeight: 0,
		BlockHash:         make(map[uint]string),
	}
	return &getter, err
}

func (opi *OPIOrdGetterTest) GetLatestBlockHeight() (uint, error) {
	return opi.LatestBlockHeight, nil
}

func (opi *OPIOrdGetterTest) GetBlockHash(blockHeight uint) (string, error) {
	if result, found := opi.BlockHash[blockHeight]; found {
		return result, nil
	}
	return "", nil
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
