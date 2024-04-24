package main

import (
	"testing"

	"github.com/RiemaLabs/modular-indexer-committee/ord"
	"github.com/RiemaLabs/modular-indexer-committee/ord/stateless"
)

func Test_SelfMint(t *testing.T) {
	stateless.SelfMintEnableHeight = 779832
	var latestHeight uint = stateless.BRC20StartHeight + ord.BitcoinConfirmations
	loadVerifyCurrentBalanceOfWallet("xordi", "bc1pkj5jjzglh99zxqu6w9vwdlpk7rqr706jw8t2jtsf4yvfrrvc6ggqlefhke", latestHeight, t, 779838)
}
