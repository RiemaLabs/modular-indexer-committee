package nubit_da

import (
	"encoding/hex"
	"strings"
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
	// NamespaceSize is the size of the hex encoded namespace string
	NamespaceSize = 29 * 2
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

func NewNubitDABackend(rpc, token, namespace string, FetchTimeout string, SubmitTimeout string) (*NubitDABackend, error) {
	client, err := proxy.NewClient(rpc, token)
	if err != nil {
		return nil, err
	}
	byteData := []byte(namespace)
	hexNamespace := hex.EncodeToString(byteData)
	fullNamespace := padNamespaceLeft(hexNamespace)
	ns, err := hex.DecodeString(fullNamespace)
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

func IsValidNamespaceID(nID string) bool {
	if len(nID) > 10 {
		return false
	}
	byteData := []byte(nID)
	hexString := hex.EncodeToString(byteData)
	return len(hexString) <= NamespaceSize
}

func padNamespaceLeft(s string) string {
	currentLength := len(s)
	if currentLength < NamespaceSize {
		return strings.Repeat("0", NamespaceSize-currentLength) + s
	}
	return s
}
