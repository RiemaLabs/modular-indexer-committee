package checkpoint

import (
	"fmt"
	"strconv"
	"strings"
)

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

type UploadRecord struct {
	Success bool
}

type UploadHistory = map[uint]map[string]UploadRecord

func NewCheckpoint(indexID *IndexerIdentification, height uint, hash string, commitment string) Checkpoint {
	blockHeight := fmt.Sprintf("%d", height)
	content := Checkpoint{
		URL:          indexID.URL,
		Name:         indexID.Name,
		Version:      indexID.Version,
		MetaProtocol: indexID.MetaProtocol,
		Height:       blockHeight,
		Hash:         hash,
		Commitment:   commitment,
	}
	return content
}

func IsValidNamespaceID(nID string) bool {
	if strings.HasPrefix(nID, "0x") {
		_, err := strconv.ParseUint(nID[2:], 16, 64)
		if err != nil {
			return false
		}
	} else {
		_, err := strconv.ParseUint(nID, 10, 64)
		if err != nil {
			return false
		}
	}
	return true
}
