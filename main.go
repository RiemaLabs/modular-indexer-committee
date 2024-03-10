package main

import (
	"sync"
	"encoding/json"
	"fmt"
	"net/http"
	"log"
	"time"
	"os"
	"strconv"

	"github.com/ethereum/go-verkle"
	"gorm.io/gorm"
	uint256 "github.com/holiman/uint256"
	"github.com/gin-gonic/gin"
)

// Define Global Varible
var BRC20StartBlock uint = 831931 // TMP FOR TEST ONLY, DELETE LATER
// var BRC20StartBlock uint = 779832
var MAXLEN int = 10

type verkleTree struct {
	element			verkle.VerkleNode
	height 			uint
	hash  			string
}

// Maintain a list of Verkle Trees
type VerkleHistory struct {
	verkleElement	[]verkleTree
	maxLen   		int
	curHeight     	uint
	sync.RWMutex
}

// NewVerkleHistory: create a VerkleHistory
func NewVerkleHistory(maxLen int, startHeight uint) *VerkleHistory {
	return &VerkleHistory{
		verkleElement: make([]verkleTree, 0, maxLen),
		maxLen:        maxLen,
		curHeight:     startHeight,
	}
}

// Push to the VerkleHistory, pop outdated Verkle Tree
func (verkles *VerkleHistory) Push(element verkleTree, increaseHeight bool) {
	verkles.RLock()
	verkleMaxLen := verkles.maxLen
	verkles.RUnlock()

	verkles.Lock()
	defer verkles.Unlock()

	if len(verkles.verkleElement) >= verkleMaxLen {
		// Remove the oldest element
		verkles.verkleElement = verkles.verkleElement[1:]
	}
	if increaseHeight {
		verkles.curHeight ++
	}
	// Append the new element
	verkles.verkleElement = append(verkles.verkleElement, element)
}

// Pop outdated element from the VerkleHostory, only used when the reorganization happens
func (verkles *VerkleHistory) PopBack() (verkleTree, bool) {
	verkles.Lock()
	defer verkles.Unlock()

	if len(verkles.verkleElement) == 0 {
		return verkleTree{}, false // When the VerkleHistory is empty, return false
	}
	element := verkles.verkleElement[0]
	verkles.verkleElement = verkles.verkleElement[1:]
	verkles.curHeight --
	return element, true
}

func (vh *VerkleHistory) PrintVerkleHistory() {
	// Used for debugging
	vh.RLock()
	defer vh.RUnlock()
	log.Println("====",len(vh.verkleElement), "====", vh.curHeight, "====")
	for _, node := range vh.verkleElement {
		log.Print(node.height, "*")
	}
}

func (verkles *VerkleHistory) createCheckpoint() (string, error){
	// get the macAddress of Indexer
	verkles.RLock()
	defer verkles.RUnlock()

	macAddress := getMACAddress()
	index := len(verkles.verkleElement) -1
	// stateRoot := verkles.verkleElement[index].element
	curHeight := verkles.verkleElement[index].height
	var curHeightString uint64 = uint64(curHeight)
	heightString := strconv.FormatUint(curHeightString, 10) 
	
	curHash := verkles.verkleElement[index].hash
	fileName := fmt.Sprintf("checkpoint-%s-BRC20-%s-%s.json", macAddress, heightString, curHash)

	// checkPoint := stateRoot.Commit() // TODO:如何生成一个checkPoint

	// create file content
	content := map[string]string{
		"indexerAPI": macAddress,
		"indexerName": "Committee1",
		"indexerVersion": "0.1",
		"metaProtocal": "BRC20",
		"latestBlockHeight": heightString,
		"latestBlockHash": curHash,
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

func (verkles *VerkleHistory) initCommittee(db *gorm.DB, stateRoot verkle.VerkleNode, latestHeight uint) {
	// no need for lock, because it is not running assynchronously
	for curHeight := BRC20StartBlock; curHeight <= latestHeight; curHeight += 1 {
		ordTransfer := getOrdTransfers(db, curHeight)
		stateRoot = processOrdTransfer(stateRoot, ordTransfer, curHeight)

		newHash, err := getBlockHash(curHeight)
		if err != nil {
			log.Println("Error getting block hash:", err)
		}
		
		verkles.Push(verkleTree{element: stateRoot, height: curHeight, hash: newHash}, false)
		verkles.curHeight = curHeight

		// send the latest verkle tree commitment
		fileName, err := verkles.createCheckpoint()
		if err != nil {
			fmt.Println("Error happens when creating file", err)
			break
		}
		uploadFile(fileName,"www.www.www.www.www") // TODO: 得到上传checkpoint的URL

		verkles.PrintVerkleHistory()
	}
}

func (verkles *VerkleHistory) checkForUpdate(db *gorm.DB) bool{
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
		index := len(verkles.verkleElement) -1
		stateRoot := verkles.verkleElement[index].element
		verkles.RUnlock()
		for curHeight := verkleCurHeight+1; curHeight <= latestBlockHeight; curHeight += 1 {
			ordTransfer := getOrdTransfers(db, curHeight)
			stateRoot = processOrdTransfer(stateRoot, ordTransfer, curHeight)
			verkles.RLock()
			newHash, err := getBlockHash(curHeight)
			verkles.RUnlock()
			if err != nil {
				log.Println("Error getting block hash:", err)
			}
			verkles.Push(verkleTree{element: stateRoot, height: curHeight, hash: newHash}, true)
			verkles.PrintVerkleHistory()
		}
		return true
	}
	return false
}

func (verkles *VerkleHistory) checkForReorg() uint{
	// check if reorgnization happened, return the number of verkle trees need to be updated
	verkles.RLock()
    defer verkles.RUnlock()

	var needToBeUpdated uint = 0
	for i := verkles.curHeight; i>=verkles.curHeight-10; i-- {
		newHash, err := getBlockHash(i)
		if err != nil {
			log.Println("Error getting block hash:", err)
		}
		index := uint(len(verkles.verkleElement)) - 1 + i - verkles.curHeight
		if verkles.verkleElement[index].hash == newHash {
			return needToBeUpdated
		} else {
			needToBeUpdated ++
			if i == verkles.curHeight-10 {
				log.Println("Critical Error")
			}
		}
	}
	return needToBeUpdated
}

func (verkles *VerkleHistory) updateCommittee(db *gorm.DB) {
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

			index := len(verkles.verkleElement) -1
			stateRoot := verkles.verkleElement[index].element
			for i := verkles.curHeight+1; i < verkles.curHeight+uint(needToBeUpdated)+1; i++ {
				ordTransfer := getOrdTransfers(db, i)
				stateRoot = processOrdTransfer(stateRoot, ordTransfer, i)
				newHash, err := getBlockHash(i)
				if err != nil {
					log.Println("Error getting block hash:", err)
				}
				verkles.Push(verkleTree{element: stateRoot, height: i, hash: newHash}, true)
			}
		}

		verkles.checkForUpdate(db)
	}
}

func main() {
	db := ConnectDatabase()
	var latestHeight uint = 831942 // TODO: 改成读一个, 目前仅测试用, 实际代码在utils.go: getMaxBlockHeight()
	stateRoot := verkle.New()
	verkles := NewVerkleHistory(MAXLEN, BRC20StartBlock)
	verkles.initCommittee(db, stateRoot, latestHeight)
	go verkles.updateCommittee(db)
	// verkles.updateCommittee(db) // Uncomment to test updateCommitteee()

	r := gin.Default()
	// // Open 3 APIs
	r.GET("/brc20_verifiable_get_current_balance_of_wallet", func(c *gin.Context) {
		verkles.RLock()
		defer verkles.RUnlock()
		tick := c.DefaultQuery("tick", "")
		newPkscript :=c.DefaultQuery("pkscript","")
		availableKey, overallKey := getHash("available-balance", tick, newPkscript), getHash("overall-balance", tick, newPkscript)
		index := len(verkles.verkleElement) -1
		stateRoot := verkles.verkleElement[index].element

		resAvail := uint256.NewInt(0)
		valueAvail, _ := stateRoot.Get(availableKey, nodeResolveFn)
		
		resOverall := uint256.NewInt(0)
		valueOverall, _ := stateRoot.Get(overallKey, nodeResolveFn)

		if len(valueAvail) == 0 && len(valueOverall) == 0{
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tick Pkscript Pair Not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"availableBalance": resAvail.SetBytes(valueAvail),
			"prevOverallBalance": resOverall.SetBytes(valueOverall),
		})
	})

	r.GET("brc20_verifiable_block_height", func(c *gin.Context){
		verkles.RLock()
		defer verkles.RUnlock()

		c.JSON(http.StatusOK, gin.H{
			"currentHeight": verkles.curHeight,
		})
	})

	r.GET("brc20_verifiable_get_current_statediff", func(c *gin.Context){
		verkles.RLock()
		defer verkles.RUnlock()

		blockheightQuery := c.DefaultQuery("blockheight", "0")

		blockheight, err := strconv.ParseUint(blockheightQuery, 10, 64)
		if err != nil {
			// Handle error, maybe return an HTTP error response
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid blockheight parameter"})
			return
		}
		stateDiff := getStateDiff(db, uint(blockheight))
		c.JSON(http.StatusOK, stateDiff)
	})


	r.Run(":8080")
}