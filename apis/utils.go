package apis

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/RiemaLabs/modular-indexer-committee/ord/getter"
)

func BatchDecodeBase64(strs []string) ([][]byte, error) {
	res := make([][]byte, 0)
	for _, s := range strs {
		bytes, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return res, err
		}
		res = append(res, bytes)
	}
	return res, nil
}

func convertMapToBRC20Event(data map[string]interface{}) (getter.BRC20Event, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal map: %v", err)
	}

	eventType, ok := data["type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid event type")
	}

	var event getter.BRC20Event
	switch eventType {
	case "deploy":
		var deployEvent getter.BRC20DeployEvent
		err = json.Unmarshal(jsonData, &deployEvent)
		event = &deployEvent
	case "mint":
		var mintEvent getter.BRC20MintEvent
		err = json.Unmarshal(jsonData, &mintEvent)
		event = &mintEvent
	case "transfer":
		var transferEvent getter.BRC20TransferEvent
		err = json.Unmarshal(jsonData, &transferEvent)
		event = &transferEvent
	case "inscribeTransfer":
		var inscribeTransferEvent getter.BRC20InscribeTransferEvent
		err = json.Unmarshal(jsonData, &inscribeTransferEvent)
		event = &inscribeTransferEvent
	default:
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %v", err)
	}

	return event, nil
}
