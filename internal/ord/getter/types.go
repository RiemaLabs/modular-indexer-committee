package getter

type OrdTransfer struct {
	ID            uint
	InscriptionID string
	OldSatpoint   string
	NewPkscript   string
	NewWallet     string
	SentAsFee     bool
	Content       []byte
	ContentType   string
}

type OrdGetter interface {
	GetLatestBlockHeight() (uint, error)
	GetBlockHash(blockHeight uint) (string, error)
	GetOrdTransfers(blockHeight uint) ([]OrdTransfer, error)
}
