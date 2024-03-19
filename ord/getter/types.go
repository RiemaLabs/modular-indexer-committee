package getter

import "github.com/RiemaLabs/indexer-committee/ord"

type OrdTransfer struct {
	ID            uint
	InscriptionID string
	OldSatpoint   string
	NewPkScript   ord.PkScript
	NewWallet     ord.Wallet
	SentAsFee     bool
	Content       []byte
	ContentType   string
}

// For the verification of light client.
type VerifiableOrdTransfer struct {
	// "" -> satPoint0 -> satPoint1 -> ...
	SatPointPath []string

	Transfer OrdTransfer
}

type OrdGetter interface {
	GetLatestBlockHeight() (uint, error)
	GetBlockHash(blockHeight uint) (string, error)
	GetOrdTransfers(blockHeight uint) ([]OrdTransfer, error)
	GetVerifiableOrdTransfers(blockHeight uint) ([]VerifiableOrdTransfer, error)
}
