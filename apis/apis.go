package apis

// import (
// 	"net/http"
// 	"strconv"

// 	"github.com/gin-gonic/gin"
// 	uint256 "github.com/holiman/uint256"
// )

// func StartService() {
// 	r := gin.Default()
// 	// // Open 3 APIs
// 	r.GET("/brc20_verifiable_get_current_balance_of_wallet", func(c *gin.Context) {
// 		verkles.RLock()
// 		defer verkles.RUnlock()
// 		tick := c.DefaultQuery("tick", "")
// 		newPkscript := c.DefaultQuery("pkscript", "")
// 		availableKey, overallKey := getHash("available-balance", tick, newPkscript), getHash("overall-balance", tick, newPkscript)
// 		index := len(verkles.verkleElement) - 1
// 		stateRoot := verkles.verkleElement[index].element

// 		resAvail := uint256.NewInt(0)
// 		valueAvail, _ := stateRoot.Get(availableKey, nodeResolveFn)

// 		resOverall := uint256.NewInt(0)
// 		valueOverall, _ := stateRoot.Get(overallKey, nodeResolveFn)

// 		if len(valueAvail) == 0 && len(valueOverall) == 0 {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "Tick Pkscript Pair Not found"})
// 			return
// 		}

// 		c.JSON(http.StatusOK, gin.H{
// 			"availableBalance":   resAvail.SetBytes(valueAvail),
// 			"prevOverallBalance": resOverall.SetBytes(valueOverall),
// 		})
// 	})

// 	r.GET("brc20_verifiable_block_height", func(c *gin.Context) {
// 		verkles.RLock()
// 		defer verkles.RUnlock()

// 		c.JSON(http.StatusOK, gin.H{
// 			"currentHeight": verkles.curHeight,
// 		})
// 	})

// 	r.GET("brc20_verifiable_get_current_statediff", func(c *gin.Context) {
// 		verkles.RLock()
// 		defer verkles.RUnlock()

// 		blockheightQuery := c.DefaultQuery("blockheight", "0")

// 		blockheight, err := strconv.ParseUint(blockheightQuery, 10, 64)
// 		if err != nil {
// 			// Handle error, maybe return an HTTP error response
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid blockheight parameter"})
// 			return
// 		}
// 		stateDiff := getStateDiff(db, uint(blockheight))
// 		c.JSON(http.StatusOK, stateDiff)
// 	})

// 	r.Run(":8080")
// }
