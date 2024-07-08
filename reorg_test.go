package main

import (
	"log"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/RiemaLabs/modular-indexer-committee/ord"
	"github.com/RiemaLabs/modular-indexer-committee/ord/stateless"
)

func Test_Serialization(t *testing.T) {
	log.Println("Test_Serialization")
	wg.Wait()
	wg.Add(1)
	defer wg.Done()
	var catchupHeight uint = 780050
	ordGetterTest, arguments := loadMain(782000)
	queue, _ := CatchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, catchupHeight)
	err := stateless.StoreHeader(queue.Header, catchupHeight)
	if err != nil {
		t.Errorf("Error storing header: %v", err)
	}
}

func Test_Deserialization(t *testing.T) {
	log.Println("Test_Deserialization")
	var catchupHeight uint = 780050 + ord.BitcoinConfirmations
	ordGetterTest, arguments := loadMain(782000)
	arguments.EnableStateRootCache = true
	// should load the generated DB
	queue2, _ := CatchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, catchupHeight)
	if queue2.Header.Height != catchupHeight {
		t.Errorf("Header height not equal")
	}
}
func Test_Recover(t *testing.T) {
	log.Println("Test_Recover")
	var catchupHeight uint = 780050 + ord.BitcoinConfirmations
	ordGetterTest, arguments := loadMain(782000)
	arguments.EnableStateRootCache = true
	queue, _ := CatchupStage(ordGetterTest, &arguments, stateless.BRC20StartHeight-1, catchupHeight)
	// stateless.StoreHeader(queue.Header, catchupHeight)
	// should load the generated DB
	queue.Recovery(ordGetterTest, catchupHeight) // this will abandon current queue
	mockService(ordGetterTest, queue, 10)        // test if queue can still grow
}

func Test_CleanPath(t *testing.T) {
	log.Println("Test_CleanPath")
	var catchupHeight uint = 780050
	cleanPath := filepath.Join(stateless.CachePath, strconv.Itoa(int(catchupHeight))+".dat")
	err := stateless.CleanPath(cleanPath)
	if err != nil {
		t.Errorf("Error cleaning path: %v", err)
	}
}
