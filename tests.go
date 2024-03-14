package main

// TODO, Unittests
// func compareServerToOPI(db *gorm.DB, endHeight uint) bool {
// 	stateRoot := verkle.New()
// 	initHeight := uint(791113)

// 	for height := initHeight; height <= endHeight; height += 1 {
// 		log.Println("[Enter height]: ", height)
// 		ordTransfer := getOrdTransfers(db, height)
// 		stateRoot = processOrdTransfer(stateRoot, ordTransfer, height)

// 		opiDeployedTicks := getDeployedTicksAtHeight(db, height)
// 		opiStateDiff := getStateDiff(db, height)

// 		for k, v := range opiStateDiff {
// 			res, _ := stateRoot.Get([]byte(k), nodeResolveFn)
// 			if len(res) == 0 {
// 				log.Println("[No such key at height] ", height)
// 				log.Println("[No such key]: ", debug_dict[k])
// 				return false
// 			}
// 			if !bytes.Equal(res, v) {
// 				log.Println("[Inconsistent at height] ", height)
// 				log.Println("[Inconsistent at key]: ", debug_dict[k])
// 				log.Println("[value from tree]: ", convertByteToInt(res))
// 				log.Println("[value from opi]: ", convertByteToInt(v))
// 				return false
// 			}
// 		}

// 		for k, v := range opiDeployedTicks {
// 			res, _ := stateRoot.Get([]byte(k), nodeResolveFn)
// 			if len(res) == 0 {
// 				log.Println("[Tick: No such key at height] ", height)
// 				log.Println("[Tick: No such key]: ", debug_dict[k])
// 				return false
// 			}
// 			if !bytes.Equal(res, v) {
// 				log.Println("[Tick: Inconsistent at height] ", height)
// 				log.Println("[Tick: Inconsistent at key]: ", debug_dict[k])
// 				log.Println("[Tick: value from tree]: ", convertByteToInt(res))
// 				log.Println("[Tick: value from opi]: ", convertByteToInt(v))
// 				return false
// 			}
// 		}
// 	}
// 	return true
// }
