package getter

import "github.com/RiemaLabs/modular-indexer-committee/ord"

// TODO: High. Record Old satpoint- Current satpoint to get OrdTransfer from the Bitcoin block directly.
type OrdTransfer struct {
	ID            uint
	InscriptionID string
	BlockHeight   uint
	OldSatpoint   string
	NewSatpoint   string
	NewPkscript   ord.Pkscript
	NewWallet     ord.Wallet
	SentAsFee     bool
	Content       []byte
	ContentType   string
	ParentID	  string
}

type OrdGetter interface {
	GetLatestBlockHeight() (uint, error)
	GetBlockHash(blockHeight uint) (string, error)
	GetOrdTransfers(blockHeight uint) ([]OrdTransfer, error)
}
