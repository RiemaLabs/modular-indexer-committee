package checkpoint

import (
	"encoding/base64"
	"strconv"

	"nubit-indexer-committee/internal/ord"
)

func NewCheckpoint(indexID IndexerIdentification, state ord.State) Checkpoint {
	blockHeight := strconv.FormatUint(uint64(state.Height), 10)
	blockHash := state.Hash
	bytes := state.Root.Commit().Bytes()
	commitment := base64.StdEncoding.EncodeToString(bytes[:])
	content := Checkpoint{
		URL:          indexID.URL,
		Name:         indexID.Name,
		Version:      indexID.Version,
		MetaProtocol: indexID.MetaProtocol,
		Height:       blockHeight,
		Hash:         blockHash,
		Commitment:   commitment,
	}
	return content
}

func UploadCheckpoint(hisotry UploadHistory, indexerID IndexerIdentification, checkpoint Checkpoint) {
	// TODO: Upload Checkpoint
}
