package checkpoint

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	sdk "github.com/RiemaLabs/nubit-da-sdk"
	"github.com/RiemaLabs/nubit-da-sdk/constant"
	"github.com/RiemaLabs/nubit-da-sdk/types"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func NewCheckpoint(indexID *IndexerIdentification, height uint, hash string, commitment string) Checkpoint {
	blockHeight := fmt.Sprintf("%d", height)
	content := Checkpoint{
		URL:          indexID.URL,
		Name:         indexID.Name,
		Version:      indexID.Version,
		MetaProtocol: indexID.MetaProtocol,
		Height:       blockHeight,
		Hash:         hash,
		Commitment:   commitment,
	}
	return content
}

func UploadCheckpointByDA(checkpoint *Checkpoint, pk, gasCode, namespaceID, network string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if network == "Pre-Alpha Testnet" {
		sdk.SetNet(constant.PreAlphaTestNet)
	} else if network == "Testnet" {
		sdk.SetNet(constant.TestNet)
	} else {
		return fmt.Errorf("unknown network: %s", network)
	}

	clientDA := sdk.NewNubit(sdk.WithCtx(ctx),
		sdk.WithGasCode(gasCode),
		sdk.WithPrivateKey(pk),
	)
	if clientDA == nil {
		return fmt.Errorf("failed to build the Nubit client")
	}

	checkpointJSON, err := json.Marshal(checkpoint)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint to JSON: %v", err)
	}

	labels := map[string]interface{}{
		"contentType": "application/json",
	}
	_, err = clientDA.UploadBytes(checkpointJSON, namespaceID, 0, labels)
	if err != nil {
		return fmt.Errorf("failed to upload checkpoint: %v", err)
	}

	return nil
}

func DownloadCheckpointByDA(namespaceID, network string, name, metaProtocol, height, hash string, runtimeOffset int, timeout time.Duration) (*Checkpoint, int, error) {
	var c Checkpoint

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if network == "Pre-Alpha Testnet" {
		sdk.SetNet(constant.PreAlphaTestNet)
	} else if network == "Testnet" {
		sdk.SetNet(constant.TestNet)
	} else {
		return nil, 0, fmt.Errorf("unknown network: %s", network)
	}

	clientDA := sdk.NewNubit(sdk.WithCtx(ctx)).Client

	resCount, err := clientDA.GetTotalDataIDsInNamesapce(ctx, &types.GetTotalDataIDsInNamesapceReq{NID: namespaceID})

	if err != nil {
		return nil, 0, fmt.Errorf("failed to get the count of data in namespace %s, error: %v", namespaceID, err)
	}

	count := int(resCount.Count)
	if count == 0 {
		return nil, 0, fmt.Errorf("the count of data in namespace %s is zero", namespaceID)
	}

	resDataIDs, err := clientDA.GetDataInNamespace(ctx, &types.GetDataInNamespaceReq{
		NID:    namespaceID,
		Limit:  100,
		Offset: runtimeOffset,
	})

	if err != nil {
		return nil, 0, fmt.Errorf("failed to get data with offset %d, in namespace %s, error: %v", runtimeOffset, namespaceID, err)
	}

	dataIDs := resDataIDs.DataIDs

	if len(dataIDs) == 0 {
		return nil, 0, fmt.Errorf("the count of data with offset %d, in namespace %s, error: %v", runtimeOffset, namespaceID, err)
	}

	for _, dataID := range dataIDs {
		runtimeOffset += 1
		datas, err := clientDA.GetData(ctx, &types.GetDataReq{
			DAID: dataID,
		})
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get data with offset %d, in namespace %s, error: %v", runtimeOffset, namespaceID, err)
		}

		decodeString, err := base64.StdEncoding.DecodeString(datas.RawData)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to parse data with offset %d, in namespace %s, error: %v", runtimeOffset, namespaceID, err)
		}
		err = json.Unmarshal(decodeString, &c)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to parse data with offset %d, in namespace %s, error: %v", runtimeOffset, namespaceID, err)
		}
		if strings.EqualFold(c.Name, name) && strings.EqualFold(c.MetaProtocol, metaProtocol) && strings.EqualFold(c.Height, height) && strings.EqualFold(c.Hash, hash) {
			return &c, runtimeOffset, nil
		}
	}
	return nil, 0, fmt.Errorf("failed to parse data with offset %d, in namespace %s, error: %v", runtimeOffset, namespaceID, err)
}

func IsValidNamespaceID(nID string) bool {
	if strings.HasPrefix(nID, "0x") {
		_, err := strconv.ParseUint(nID[2:], 16, 64)
		if err != nil {
			return false
		}
	} else {
		_, err := strconv.ParseUint(nID, 10, 64)
		if err != nil {
			return false
		}
	}

	return true
}

func CreateNamespace(pk, gasCode, namespaceName, network string) (string, error) {
	ctx := context.TODO()
	if network == "Pre-Alpha Testnet" {
		sdk.SetNet(constant.PreAlphaTestNet)
	} else if network == "Testnet" {
		sdk.SetNet(constant.TestNet)
	} else {
		return "", fmt.Errorf("unknown network: %s", network)
	}

	clientDA := sdk.NewNubit(sdk.WithCtx(ctx),
		sdk.WithGasCode(gasCode),
		sdk.WithPrivateKey(pk),
	)
	if clientDA == nil {
		return "", fmt.Errorf("failed to build the Nubit client")
	}
	ns, err := clientDA.CreateNamespace(namespaceName, "Private", "", []string{})
	if err != nil {
		return "", err
	}

	// Wait for the new block.
	time.Sleep(time.Second * 25)

	tx, err := clientDA.Client.GetTransaction(ctx, &types.GetTransactionReq{
		TxID: ns.TxID,
	})

	if err != nil {
		return "", err
	}

	return tx.NID, err
}

func UploadCheckpointByS3(c *Checkpoint, accessKey, secretKey, region, bucket string, timeout time.Duration) error {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithRegion(region),
	)
	if err != nil {
		return fmt.Errorf("failed to create aws config, error: %v", err)
	}

	var awsS3Client = s3.NewFromConfig(cfg)
	uploader := manager.NewUploader(awsS3Client)

	objectKey := fmt.Sprintf("checkpoint-%s-%s-%s-%s.json", c.Name, c.MetaProtocol, c.Height, c.Hash)

	checkpointJSON, err := json.Marshal(c)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := uploader.Upload(ctx, &s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(objectKey),
			Body:   bytes.NewReader(checkpointJSON),
		})
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			log.Printf("Checkpoint %s uploaded to S3 successfully!", objectKey)
			return nil
		} else {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}
}

func DownloadCheckpointByS3(region, bucket string, name, metaProtocol, height, hash string, timeout time.Duration) (*Checkpoint, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	bytes := make([]byte, 0)
	writer := manager.NewWriteAtBuffer(bytes)

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create aws config, error: %v", err)
	}
	var awsS3Client = s3.NewFromConfig(cfg)
	downloader := manager.NewDownloader(awsS3Client)

	objectKey := fmt.Sprintf("checkpoint-%s-%s-%s-%s.json", name, metaProtocol, height, hash)

	numBytes, err := downloader.Download(ctx, writer, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download file with bytes: %d, error: %v", numBytes, err)
	}

	var c Checkpoint

	err = json.Unmarshal(bytes, &c)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file, error: %v", err)
	}

	return &c, nil
}
