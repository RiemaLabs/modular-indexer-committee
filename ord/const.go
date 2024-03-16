package ord

import verkle "github.com/ethereum/go-verkle"

// The first block height of the brc-20 protocol.
const BRC20StartHeight uint = 779832

// The number of confirmations to be considered immutable and can't be re-organized.
const BitcoinConfirmations uint = 6

var nodeResolveFn verkle.NodeResolverFn = nil
