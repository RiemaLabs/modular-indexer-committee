package checkpoint

type IndexerIdentification struct {
	URL          string
	Name         string
	Version      string
	MetaProtocol string
}

type Checkpoint struct {
	URL          string
	Name         string
	Version      string
	MetaProtocol string
	Height       string
	Hash         string
	Commitment   string
}

type UploadHistory = map[uint]map[string]bool
