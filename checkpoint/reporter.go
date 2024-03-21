package checkpoint

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	sdk "github.com/RiemaLabs/nubit-da-sdk"

	"github.com/RiemaLabs/indexer-committee/ord/stateless"
	"github.com/RiemaLabs/nubit-da-sdk/constant"
	"github.com/RiemaLabs/nubit-da-sdk/types"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
)

func NewCheckpoint(indexID IndexerIdentification, header stateless.Header) Checkpoint {
	blockHeight := strconv.FormatUint(uint64(header.Height), 10)
	blockHash := header.Hash
	bytes := header.Root.Commit().Bytes()
	commitment := base64.StdEncoding.EncodeToString(bytes[:])
	content := Checkpoint{
		URL:          indexID.URL,
		Name:         indexID.Name,
		Version:      indexID.Version,
		MetaProtocol: indexID.MetaProtocol,
		Height:       blockHeight,
		Hash:         blockHash,
		Commitment:   commitment,
	}
	return content
}

var awsS3Client *s3.Client

func UploadCheckpointByS3(indexerID IndexerIdentification, checkpoint Checkpoint, region, bucket string, timeout time.Duration) error {
	// the SDK uses its default credential chain to find AWS credentials. This default credential chain looks for credentials in the following order:aws.Configconfig.LoadDefaultConfig
	// creds := credentials.NewStaticCredentialsProvider(your_access_key, your_secret_key, "")
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return err
	}

	awsS3Client = s3.NewFromConfig(cfg)
	uploader := manager.NewUploader(awsS3Client)

	objectKey := fmt.Sprintf("test/checkpoint-%s-%s-%s-%s.json",
		checkpoint.Name, checkpoint.MetaProtocol, checkpoint.Height, checkpoint.Hash)

	// change format into JSON
	checkpointJSON, err := json.Marshal(checkpoint)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
		Body:   bytes.NewReader(checkpointJSON),
	})

	return err
}

// TODO: Urgent. Move the createNamespace to the main process.
// Don't use the hardcode address.
// Only need to create namespace once.
func UploadCheckpointByDA(indexerID IndexerIdentification, checkpoint Checkpoint, daRPC, pk, inviteCode string, timeout time.Duration) error {
	// change format into JSON
	checkpointJSON, err := json.Marshal(checkpoint)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	sdk.SetNet(constant.TestNet)

	// TODO: Medium. Allow to setup the SDK without InviteCode.
	clientDA := sdk.NewNubit(sdk.WithCtx(ctx),
		sdk.WithRpc(daRPC),
		sdk.WithInviteCode(inviteCode),
		sdk.WithPrivateKey(pk),
	)
	if clientDA == nil {
		return fmt.Errorf("cannot build the Nubit client")
	}

	ns, err := clientDA.CreateNamespace("test", "Private", "mpVLaLbmMEeKL8snmQjaXVetUe73ugqRru", []string{"mpVLaLbmMEeKL8snmQjaXVetUe73ugqRru", "mnj48QUBZr8YvRXkgTCCCeRLRkq295LAoK"})
	if err != nil {
		log.Fatalf("Failed to create namespace: %v\n", err)
	}
	fmt.Println("\n\n namespace---:", ns)

	time.Sleep(time.Second * 22)
	tx, err := clientDA.Client.GetTransaction(ctx, &types.GetTransactionReq{
		TxID: ns.TxID,
	})
	if err != nil {
		return err
	}
	fmt.Println("\n\n transaction:", tx)

	labels := map[string]interface{}{
		"contentType": "application/json",
	}
	upload, err := clientDA.UploadBytes(checkpointJSON, tx.NID, 0, labels)
	if err != nil {
		fmt.Println("Failed to upload checkpoint:", err)
		return err
	}
	fmt.Println("\n upload:", upload)

	namespaces, err := clientDA.Client.GetNamespaces(ctx, &types.GetNamespacesReq{Limit: 50, Offset: 0, Filter: struct {
		Owner string `json:"owner,omitempty"`
		Admin string `json:"admin,omitempty"`
	}{
		Owner: "mpVLaLbmMEeKL8snmQjaXVetUe73ugqRru",
	}})
	if err != nil {
		return err
	}

	time.Sleep(time.Second * 22)
	var Nss []string
	if len(namespaces.Namespaces) > 0 {
		for _, ns := range namespaces.Namespaces {
			fmt.Println("namespace:", ns.NamespaceID)
			Nss = append(Nss, ns.NamespaceID)
		}
	}
	fmt.Println("namespace:", Nss)

	datas, err := clientDA.Client.GetDatas(ctx, &types.GetDatasReq{
		NID:         Nss,
		BlockNumber: 0,
	})
	if err != nil {
		return err
	}
	marshal, err := json.Marshal(datas)
	if err != nil {
		return err
	}
	fmt.Println("\n datas:", string(marshal))
	return nil
}
