package getter

// TODO: Use the Bitcoin Block directly
type BitcoinOrdGetter struct {
}

// type RPCRequest struct {
// 	JSONRPC string      `json:"jsonrpc"`
// 	ID      int         `json:"id"`
// 	Method  string      `json:"method"`
// 	Params  interface{} `json:"params"`
// }

// type RPCResponse struct {
// 	Result json.RawMessage `json:"result"`
// 	Error  interface{}     `json:"error"`
// 	ID     int             `json:"id"`
// }

// func FetchBlockHeight(config Config) (uint, error) {
// 	request := RPCRequest{
// 		JSONRPC: "1.0",
// 		ID:      1,
// 		Method:  "getblockcount",
// 	}
// 	requestBody, err := json.Marshal(request)
// 	if err != nil {
// 		return 0, fmt.Errorf("error marshalling request: %w", err)
// 	}

// 	// Send request
// 	client := &http.Client{}
// 	req, err := http.NewRequest("POST", config.BitcoinRPC.URL, bytes.NewBuffer(requestBody))
// 	if err != nil {
// 		return 0, fmt.Errorf("error creating HTTP request: %w", err)
// 	}
// 	req.Header.Set("Content-Type", "application/json")
// 	req.SetBasicAuth(config.BitcoinRPC.Username, config.BitcoinRPC.Password)

// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return 0, fmt.Errorf("error sending request: %w", err)
// 	}
// 	defer resp.Body.Close()

// 	// Resolve response
// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return 0, fmt.Errorf("error reading response body: %w", err)
// 	}

// 	var response RPCResponse
// 	err = json.Unmarshal(body, &response)
// 	if err != nil {
// 		return 0, fmt.Errorf("error unmarshalling response: %w", err)
// 	}

// 	if response.Error != nil {
// 		return 0, fmt.Errorf("RPC error: %v", response.Error)
// 	}

// 	// Parse the block count
// 	var blockCount uint
// 	err = json.Unmarshal(response.Result, &blockCount)
// 	if err != nil {
// 		return 0, fmt.Errorf("error unmarshalling result: %w", err)
// 	}

// 	return blockCount, nil
// }
