package nubit_da

import (
	"encoding/hex"
	"time"

	"github.com/rollkit/go-da"
	"github.com/rollkit/go-da/proxy"
)

const (
	// The URL of Nubit DA node
	NodeRPCFlagName = "da.node_rpc"
	// The auth token of Nubit DA node
	AuthTokenFlagName = "da.auth_token"
	// The namespace of running Layer 2
	NamespaceFlagName = "da.namespace"
	// Default Namespace for okx-brc20
	DefaultNamespace = "00000000000000000000000000000000000000006F6B782D6272633230"
	// Default local deployed Nubit Node
	DefaultNodeRPC = "http://localhost:26658"
	// 30*time.Duration(l.RollupConfig.BlockTime)*time.Second
	DefaultFetchTimeout = time.Minute
	// 30*time.Duration(l.RollupConfig.BlockTime)*time.Second
	DefaultSubmitTimeout = time.Minute
	// Prefix
	NubitDataPrefix = 0xda
)


type NubitDABackend struct {
	Client       da.DA
	FetchTimeout time.Duration

	SubmitTimeout time.Duration
	Namespace     da.Namespace
}

func NewNubitDABackend(rpc, token, FetchTimeout string, SubmitTimeout string) (*NubitDABackend, error) {
	client, err := proxy.NewClient(rpc, token)
	if err != nil {
		return nil, err
	}
	ns, err := hex.DecodeString(DefaultNamespace)
	if err != nil {
		return nil, err
	}

	transFetchTimeout, err := time.ParseDuration(FetchTimeout)
	if err != nil {
		transFetchTimeout = DefaultFetchTimeout
	}

	transSubmitTimeout, err := time.ParseDuration(SubmitTimeout)
	if err != nil {
		transSubmitTimeout = DefaultSubmitTimeout
	}

	return &NubitDABackend{
		Client:        client,
		FetchTimeout:  transFetchTimeout,
		SubmitTimeout: transSubmitTimeout,
		Namespace:     ns,
	}, nil
}

// func NewNubitDABackendFromCfg(c CLIConfig) (*NubitDABackend, error) {
// 	return NewNubitDABackend(c.Rpc, c.AuthToken, c.Namespace, c.FetchTimeout, c.SubmitTimeout)
// }
