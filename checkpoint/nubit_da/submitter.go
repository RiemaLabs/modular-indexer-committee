package nubit_da

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"log"

	"github.com/RiemaLabs/modular-indexer-committee/checkpoint"
)

func UploadCheckpointByDA(c *checkpoint.Checkpoint, nodeRpc string, authToken string, fetchTimeout string, submitTimeout string) error {
	checkpointJSON, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to generate checkpoint, err: %v", err)
	}
	nubitDABackend, err := NewNubitDABackend(nodeRpc, authToken, fetchTimeout, submitTimeout)
	if err != nil {
		return fmt.Errorf("failed to connect to Nubit DA, err: %v", err)
	}
	log.Println("building Calldata transaction candidate", "size", len(checkpointJSON))
	ctx, cancel := context.WithTimeout(context.Background(), nubitDABackend.SubmitTimeout)
	ids, err := nubitDABackend.Client.Submit(ctx, [][]byte{checkpointJSON}, -1, nubitDABackend.Namespace)
	cancel()
	if err == nil && len(ids) == 1 {
		log.Println("🏆 nubit: blob successfully submitted", "id", hex.EncodeToString(ids[0]))
	} else {
		log.Println("❗ nubit: blob submission failed", "err", err)
	}
	return nil
}
