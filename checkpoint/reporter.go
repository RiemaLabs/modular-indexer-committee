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

func UploadCheckpoint(history UploadHistory, indexerID IndexerIdentification, checkpoint Checkpoint, region string, bucket string) {
	// the SDK uses its default credential chain to find AWS credentials. This default credential chain looks for credentials in the following order:aws.Configconfig.LoadDefaultConfig
	// creds := credentials.NewStaticCredentialsProvider(your_access_key, your_secret_key, "")
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		log.Fatal(err)
	}

	awsS3Client = s3.NewFromConfig(cfg)
	uploader := manager.NewUploader(awsS3Client)

	objectKey := fmt.Sprintf("test/checkpoint-%s-%s-%s-%s.json",
		checkpoint.Name, checkpoint.MetaProtocol, checkpoint.Height, checkpoint.Hash)

	// change format into JSON
	checkpointJSON, err := json.Marshal(checkpoint)
	if err != nil {
		log.Printf("Failed to marshal checkpoint to JSON: %v\n", err)
		return
	}

	// locate history map
	heightUint, err := strconv.ParseUint(checkpoint.Height, 10, 64)
	if err != nil {
		log.Printf("Failed to convert checkpoint height to uint64: %v\n", err)
		return
	}
	if _, ok := history[uint(heightUint)]; !ok {
		history[uint(heightUint)] = make(map[string]bool)
	}

	output, err := uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
		Body:   bytes.NewReader(checkpointJSON),
	})

	if err != nil {
		log.Printf("Checkpoint upload failed: %v\n", err)
		history[uint(heightUint)][objectKey] = false
	} else {
		log.Println("Checkpoint uploaded successfully to S3!")
		log.Println("Upload output:", output)
		history[uint(heightUint)][objectKey] = true
	}
}

// TODO: upload to DA
func UploadCheckpointDA(indexerID IndexerIdentification, checkpoint Checkpoint, region string, bucket string) {
	// change format into JSON
	checkpointJSON, err := json.Marshal(checkpoint)
	if err != nil {
		log.Printf("Failed to marshal checkpoint to JSON: %v\n", err)
		return
	}

	ctx := context.Background()
	sdk.SetNet(constant.TestNet)
	clientDA := sdk.NewNubit(sdk.WithCtx(ctx),
		sdk.WithRpc("https://test.api.nubit.network:444"),
		sdk.WithInviteCode("7mkEPWPBBrMr12WKNsL2UALvqYfbox"),
		sdk.WithPrivateKey("7ae9984540c0a3bb8d5a627010601d4529c276e526e08b136d1c24e5c72195df"))
	if clientDA == nil {
		panic("clientDA is nil")
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
		fmt.Println(err)
		return
	}
	fmt.Println("\n\n transaction:", tx)

	labels := map[string]interface{}{
		"contentType": "application/json",
		"customLabel": "value",
	}
	upload, err := clientDA.UploadBytes(checkpointJSON, tx.NID, 0, labels)
	if err != nil {
		fmt.Println("Failed to upload checkpoint:", err)
		return
	}
	fmt.Println("\n upload:", upload)

	// 获取命名空间列表
	namespaces, err := clientDA.Client.GetNamespaces(ctx, &types.GetNamespacesReq{Limit: 50, Offset: 0, Filter: struct {
		Owner string `json:"owner,omitempty"`
		Admin string `json:"admin,omitempty"`
	}{
		Owner: "mpVLaLbmMEeKL8snmQjaXVetUe73ugqRru",
	}})
	if err != nil {
		return
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

	// 获取数据
	datas, err := clientDA.Client.GetDatas(ctx, &types.GetDatasReq{
		NID:         Nss,
		BlockNumber: 0,
	})
	if err != nil {
		return
	}
	marshal, err := json.Marshal(datas)
	if err != nil {
		return
	}
	fmt.Println("\n datas:", string(marshal))
}
