package getter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/RiemaLabs/modular-indexer-committee/internal/metrics"
)

type DatabaseConfig struct {
	EventUrl string
	HashUrl  string
}

type OKXBRC20Getter struct {
	eventUrl string
	hashUrl  string
}

func NewOKXBRC20Getter(config *DatabaseConfig) (*OKXBRC20Getter, error) {
	eventUrl, hashUrl := config.EventUrl, config.HashUrl
	if eventUrl == "" {
		return nil, fmt.Errorf("empty Event Url")
	}
	if hashUrl == "" {
		return nil, fmt.Errorf("empty Hash Url")
	}
	return &OKXBRC20Getter{
		eventUrl: eventUrl,
		hashUrl:  hashUrl,
	}, nil
}

func (okx *OKXBRC20Getter) GetLatestBlockHeight() (uint, error) {
	defer metrics.ObserveDBQuery("getLatestBlockHeight", time.Now())
	url := okx.eventUrl + "/api/v1/node/info"
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	chainInfo, ok := result["data"].(map[string]interface{})["chainInfo"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("missing chainInfo field")
	}

	ordBlockHeight, ok := chainInfo["ordBlockHeight"].(float64)
	if !ok {
		return 0, fmt.Errorf("missing ordBlockHeight field")
	}

	return uint(ordBlockHeight), nil
}

func (okx *OKXBRC20Getter) GetBlockHash(blockHeight uint) (string, error) {
	defer metrics.ObserveDBQuery("getBlockHash", time.Now())

	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "getblockhash",
		"params":  []uint{blockHeight},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(okx.hashUrl, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	if res["error"] != nil {
		return "", fmt.Errorf("error: %v", res["error"])
	}

	if result, ok := res["result"].(string); ok {
		return result, nil
	}

	return "", fmt.Errorf("unexpected response format")
}

func (okx *OKXBRC20Getter) GetOrdTransfers(blockHeight uint) ([]BRC20Event, error) {
	defer metrics.ObserveDBQuery("getOrdTransfers", time.Now())
	blockHash, err := okx.GetBlockHash(blockHeight)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/api/v1/brc20/block/%s/events", okx.eventUrl, blockHash)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error: received non-200 status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Block []struct {
				Events []json.RawMessage `json:"events"`
			} `json:"block"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("error: %s", result.Msg)
	}

	var events []BRC20Event
	for _, block := range result.Data.Block {
		for _, rawEvent := range block.Events {
			var base BaseEvent
			if err := json.Unmarshal(rawEvent, &base); err != nil {
				return nil, err
			}

			var event BRC20Event
			switch base.EventType {
			case "deploy":
				var deployEvent BRC20DeployEvent
				if err := json.Unmarshal(rawEvent, &deployEvent); err != nil {
					return nil, err
				}
				deployEvent.BlockHeight = blockHeight
				event = &deployEvent
			case "mint":
				var mintEvent BRC20MintEvent
				if err := json.Unmarshal(rawEvent, &mintEvent); err != nil {
					return nil, err
				}
				mintEvent.BlockHeight = blockHeight
				event = &mintEvent
			case "transfer":
				var transferEvent BRC20TransferEvent
				if err := json.Unmarshal(rawEvent, &transferEvent); err != nil {
					return nil, err
				}
				transferEvent.BlockHeight = blockHeight
				event = &transferEvent
			case "inscribeTransfer":
				var inscribeTransferEvent BRC20InscribeTransferEvent
				if err := json.Unmarshal(rawEvent, &inscribeTransferEvent); err != nil {
					return nil, err
				}
				inscribeTransferEvent.BlockHeight = blockHeight
				event = &inscribeTransferEvent
			default:
				continue
			}

			events = append(events, event)
		}
	}

	return events, nil
}
