package apis

type Brc20VerifiableGetCurrentBalanceOfWalletRequest struct {
	Tick     string `json:"tick"`
	Pkscript string `json:"pkscript"`
}

type Brc20VerifiableGetCurrentBalanceOfWalletResponse struct {
	Error  string `json:"error"`
	Result string `json:"result"`
	Proof  string `json:"proof"`
}

type Brc20VerifiableLatestStateProofResponse struct {
	Keys       []string `json:"keys"`
	PreValues  []string `json:"preValues"`
	PostValues []string `json:"postValues"`
	Proof      string   `json:"proof"`
}
