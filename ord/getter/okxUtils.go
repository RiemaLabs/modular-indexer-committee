package getter

import (
	"bytes"
	"log"
	"fmt"
	"net/http"
	"time"
)

var maxRetries = 5

func getOrdTransferWithRetries(url string, maxRetries int, blockHeight uint) (*http.Response, error) {
	var resp *http.Response
	var err error

	for i := 0; i < maxRetries; i++ {
		resp, err = http.Get(url)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return resp, nil
		}

		resp.Body.Close()
		time.Sleep(1 * time.Second)
		log.Printf("Retrying to get OrdTransfer at Height: %d, Attempt: %d of %d\n", blockHeight, i+1, maxRetries)
	}

	if err == nil {
		err = fmt.Errorf("error: received non-200 status code %d", resp.StatusCode)
	}
	return nil, err
}

func getHeightWithRetries(client *http.Client, url string, maxRetries int) (*http.Response, error) {
	var resp *http.Response
	var err error

	for i := 0; i < maxRetries; i++ {
		resp, err = client.Get(url)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return resp, nil
		}

		resp.Body.Close()
		time.Sleep(1 * time.Second)
		log.Printf("Retrying to get Latest Height %d times\n", i+1)
	}

	if err == nil {
		err = fmt.Errorf("error: received non-200 status code %d", resp.StatusCode)
	}
	return nil, err
}

func getHashWithRetries(url string, blockHeight uint, contentType string, data []byte, maxRetries int) (*http.Response, error) {
	var resp *http.Response
	var err error

	for i := 0; i < maxRetries; i++ {
		resp, err = http.Post(url, contentType, bytes.NewBuffer(data))
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return resp, nil
		}

		resp.Body.Close()
		time.Sleep(1 * time.Second)
		log.Printf("Retrying to get BlockHash at Height: %d, Attempt: %d of %d\n", blockHeight, i+1, maxRetries)
	}

	return nil, err
}
