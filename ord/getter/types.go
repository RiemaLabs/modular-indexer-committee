package getter

import "github.com/RiemaLabs/indexer-committee/ord"

// The OrdTransfer defined by OPI.
// For the internal computation.
type OrdTransfer struct {
	ID            uint
	InscriptionID string
	OldSatpoint   string
	NewPkscript   ord.Pkscript
	NewWallet     ord.Wallet
	SentAsFee     bool
	Content       []byte
	ContentType   string

	// BRC-20 Special
	TransferInscribeDone     bool
	TransferInscribePkscript ord.Pkscript
	TransferInscribeWallet   ord.Wallet
	TransferTransferDone     bool
}

// The verifiableOrdTransfer.
// For the verification.
type VerifiableOrdTransfer struct {
	ordTransfer  OrdTransfer
	satPointPath []string
}

type OrdGetter interface {
	GetLatestBlockHeight() (uint, error)
	GetBlockHash(blockHeight uint) (string, error)
	GetOrdTransfers(blockHeight uint) ([]OrdTransfer, error)
	GetVerifiableOrdTransfers(blockHeight uint) ([]VerifiableOrdTransfer, error)
}
