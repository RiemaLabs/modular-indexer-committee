package ord

type TXID string

type Wallet string
type Pkscript string

// Example: 521f8eccffa4c41a3a7728dd012ea5a4a02feed81f41159231251ecf1e5c79dai0
// The part in front of the i is the transaction ID (txid) of the reveal transaction.
// The number after the i defines the index (starting at 0) of new inscriptions being inscribed in the reveal transaction.
type InscriptionID string

// Example: 680df1e4d43016571e504b0b142ee43c5c0b83398a97bdcfd94ea6f287322d22:0
// An outpoint consists of a transaction ID and output index.
type OutPoint struct {
	txID   TXID
	offset uint64
}

// A satpoint may be used to indicate the location of a sat within an output.
// A satpoint consists of an outpoint with the addition of the offset of the ordinal within that output.
// For example, if the sat in question is at offset 6 in the first output of a transaction, its satpoint is:
// 680df1e4d43016571e504b0b142ee43c5c0b83398a97bdcfd94ea6f287322d22:0:6

type SatPoint struct {
	outPoint OutPoint
	offset   uint64
}
