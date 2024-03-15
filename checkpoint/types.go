package checkpoint

type IndexerIdentification struct {
	URL          string
	Name         string
	Version      string
	MetaProtocol string
}

// CheckpointFromCommitteeIndexer
type Checkpoint struct {
	// Hex of the Commitment of the Verkle Tree Root
	Commitment string `json:"commitment"`
	// Hex of the BlockHash of the checkpoint
	Hash string `json:"hash"`
	// BlockHeight of the checkpoint
	Height string `json:"height"`
	// Protocol name used by the indexer, fixed as "BRC-20" now
	MetaProtocol string `json:"metaProtocol"`
	// Name of the indexer
	Name string `json:"name"`
	// URL of the indexer service
	URL string `json:"url"`
	// Version number of the Modular Indexer
	Version string `json:"version"`
}

type UploadHistory = map[uint]map[string]bool
