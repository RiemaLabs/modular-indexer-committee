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
	// loadVerifyCurrentBalanceOfWallet("ordi", "bc1pxaneaf3w4d27hl2y93fuft2xk6m4u3wc4rafevc6slgd7f5tq2dqyfgy06", latestHeight, t)
}
