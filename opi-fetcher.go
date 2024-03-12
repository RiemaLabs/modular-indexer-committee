package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	verkle "github.com/ethereum/go-verkle"
	"gorm.io/gorm"
)

type State struct {
	root   verkle.VerkleNode
	height uint
	hash   string
}

// Historical state neither uploads Checkpoint nor records hash.
func (state *State) HasHash() bool {
	return state.hash != ""
}

// Maintain a queue of Verkle Trees
type VerkleSlots struct {
	verkleElement []State
	maxLen        uint
	sync.RWMutex
}

// InitVerkleSlots: create the queue of the verkle trees.

func InitVerkleSlotsFromScratch(db *gorm.DB) *VerkleSlots {
	maxLen := BitcoinConfirmations
	startHeight := BRC20StartHeight
	slots := make([]State, 0, maxLen)
	stateRoot := verkle.New()

	for i := startHeight; i < startHeight+maxLen; i++ {
		ordTransfer := getOrdTransfers(db, i)
		stateRoot = processOrdTransfer(stateRoot, ordTransfer, i)
		hash := ""
		newHash, err := getBlockHash(i)
		if err != nil {
			log.Println("Error getting block hash:", err)
		}
		verkles.Push(State{root: stateRoot, height: i, hash: newHash}, true)
	}
	return VerkleSlots{
		verkleElement: slots,
		maxLen:        maxLen,
	}
}

func (verkles *VerkleSlots) check() {
	if len(verkles.verkleElement) != int(verkles.maxLen) {
		log.Fatalf("Unmatched verkle slots length! Current: %s", len(verkles.verkleElement))
	}
}

func (verkles *VerkleSlots) StartHeight() uint {
	return verkles.verkleElement[0].height
}

func (verkles *VerkleSlots) LastestHeight() uint {
	return verkles.StartHeight() + verkles.maxLen
}

// Offer the latest verkleTree and pop the oldest verkleTree.
func (verkles *VerkleSlots) Offer(element State) {
	verkles.Lock()
	defer verkles.Unlock()
	verkles.verkleElement = verkles.verkleElement[1:]
	verkles.verkleElement = append(verkles.verkleElement, element)
	verkles.check()
}

// Pop outdated element from the VerkleHostory, only used when the reorganization happens
func (verkles *VerkleSlots) PopBack() (State, bool) {
	verkles.Lock()
	defer verkles.Unlock()

	if len(verkles.verkleElement) == 0 {
		return State{}, false // When the VerkleHistory is empty, return false
	}
	element := verkles.verkleElement[0]
	verkles.verkleElement = verkles.verkleElement[1:]
	verkles.curHeight--
	return element, true
}

func (vh *VerkleSlots) PrintVerkleHistory() {
	// Used for debugging
	vh.RLock()
	defer vh.RUnlock()
	log.Println("====", len(vh.verkleElement), "====", vh.curHeight, "====")
	for _, node := range vh.verkleElement {
		log.Print(node.height, "*")
	}
}

func (verkles *VerkleSlots) createCheckpoint() (string, error) {
	// get the macAddress of Indexer
	verkles.RLock()
	defer verkles.RUnlock()

	macAddress := getMACAddress()
	index := len(verkles.verkleElement) - 1
	// stateRoot := verkles.verkleElement[index].element
	curHeight := verkles.verkleElement[index].height
	var curHeightString uint64 = uint64(curHeight)
	heightString := strconv.FormatUint(curHeightString, 10)

	curHash := verkles.verkleElement[index].hash
	fileName := fmt.Sprintf("checkpoint-%s-BRC20-%s-%s.json", macAddress, heightString, curHash)

	// checkPoint := stateRoot.Commit() // TODO:如何生成一个checkPoint

	// create file content
	content := map[string]string{
		"indexerAPI":        macAddress,
		"indexerName":       "Committee1",
		"indexerVersion":    "0.1",
		"metaProtocal":      "BRC20",
		"latestBlockHeight": heightString,
		"latestBlockHash":   curHash,
		"indexerCommitment": "",
	}

	// create and open file
	file, err := os.Create(fileName)
	if err != nil {
		fmt.Println("Error happens when creating file", err)
		return "", err
	}

	// write into the file
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(content); err != nil {
		fmt.Println("Error happens when writing to JSON:", err)
		file.Close()
		return "", err
	}

	// close the file
	if err := file.Close(); err != nil {
		fmt.Println("Error happens when closing file:", err)
		return "", err
	}
	return fileName, nil

}

func (verkles *VerkleSlots) initCommittee(db *gorm.DB, stateRoot verkle.VerkleNode, latestHeight uint) {
	// no need for lock, because it is not running assynchronously
	for curHeight := BRC20StartHeight; curHeight <= latestHeight; curHeight += 1 {
		ordTransfer := getOrdTransfers(db, curHeight)
		stateRoot = processOrdTransfer(stateRoot, ordTransfer, curHeight)

		newHash, err := getBlockHash(curHeight)
		if err != nil {
			log.Println("Error getting block hash:", err)
		}

		verkles.Push(State{root: stateRoot, height: curHeight, hash: newHash}, false)
		verkles.curHeight = curHeight

		// send the latest verkle tree commitment
		fileName, err := verkles.createCheckpoint()
		if err != nil {
			fmt.Println("Error happens when creating file", err)
			break
		}
		uploadFile(fileName, "www.www.www.www.www") // TODO: 得到上传checkpoint的URL

		verkles.PrintVerkleHistory()
	}
}

func (verkles *VerkleSlots) checkForUpdate(db *gorm.DB) bool {
	latestBlockHeight, err := getMaxBlockHeight()
	if err != nil {
		log.Printf("Error finding max block_height: %v", err)
	}
	verkles.RLock()
	verkleCurHeight := verkles.curHeight
	verkles.RUnlock()
	if latestBlockHeight > verkleCurHeight {
		log.Println("Getting new blocks at Height", latestBlockHeight)
		verkles.RLock()
		index := len(verkles.verkleElement) - 1
		stateRoot := verkles.verkleElement[index].root
		verkles.RUnlock()
		for curHeight := verkleCurHeight + 1; curHeight <= latestBlockHeight; curHeight += 1 {
			ordTransfer := getOrdTransfers(db, curHeight)
			stateRoot = processOrdTransfer(stateRoot, ordTransfer, curHeight)
			verkles.RLock()
			newHash, err := getBlockHash(curHeight)
			verkles.RUnlock()
			if err != nil {
				log.Println("Error getting block hash:", err)
			}
			verkles.Push(State{root: stateRoot, height: curHeight, hash: newHash}, true)
			verkles.PrintVerkleHistory()
		}
		return true
	}
	return false
}

func (verkles *VerkleSlots) checkForReorg() uint {
	// check if reorgnization happened, return the number of verkle trees need to be updated
	verkles.RLock()
	defer verkles.RUnlock()

	var needToBeUpdated uint = 0
	for i := verkles.curHeight; i >= verkles.curHeight-10; i-- {
		newHash, err := getBlockHash(i)
		if err != nil {
			log.Println("Error getting block hash:", err)
		}
		index := uint(len(verkles.verkleElement)) - 1 + i - verkles.curHeight
		if verkles.verkleElement[index].hash == newHash {
			return needToBeUpdated
		} else {
			needToBeUpdated++
			if i == verkles.curHeight-10 {
				log.Println("Critical Error")
			}
		}
	}
	return needToBeUpdated
}

func (verkles *VerkleSlots) updateCommittee(db *gorm.DB) {
	for {
		time.Sleep(10 * time.Second) // Check for update every 10 seconds

		needToBeUpdated := verkles.checkForReorg()
		if needToBeUpdated > 10 {
			log.Println("Critical Error")
			os.Exit(-1)
		} else if needToBeUpdated < 10 && needToBeUpdated > 0 {
			for i := uint(0); i < needToBeUpdated; i++ {
				_, ok := verkles.PopBack()
				if !ok {
					log.Println("Error popping back")
				}
			}

			index := len(verkles.verkleElement) - 1
			stateRoot := verkles.verkleElement[index].root
			for i := verkles.curHeight + 1; i < verkles.curHeight+uint(needToBeUpdated)+1; i++ {
				ordTransfer := getOrdTransfers(db, i)
				stateRoot = processOrdTransfer(stateRoot, ordTransfer, i)
				newHash, err := getBlockHash(i)
				if err != nil {
					log.Println("Error getting block hash:", err)
				}
				verkles.Push(State{root: stateRoot, height: i, hash: newHash}, true)
			}
		}

		verkles.checkForUpdate(db)
	}
}
