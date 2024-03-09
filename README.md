# Committee Indexer Demo
Collecting the latest State using Verkle Tree as a committee indexer

## Collecting From the First Brc20 blocks
func `initCommittee` initialize the verkle tree since the lowest block

## Polling From the latest OPI database
func `updateCommittee` is a non-return function, it will collect the latest state of the current OPI database. There will be only two conditions here:

1. There is a new level collected from the OPI, the latest state needs to be collected and then be used to update the verkle tree.
Func `newheight` take care of this.
2. There is a reorgnization happening at the blockchain, the verkle tree need to go back to the begginning of the reorgnization and update correspondingly. Func `reorg` take care of this.

The corresponding new conditions will be detected by the function `checkLatestState`.

Once the update is down, the latest verkle tree should be uploaded to the AWS.