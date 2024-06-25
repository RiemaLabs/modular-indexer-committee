package aws_s3

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	sdk "github.com/RiemaLabs/nubit-da-sdk"
	"github.com/RiemaLabs/nubit-da-sdk/constant"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/RiemaLabs/modular-indexer-committee/checkpoint"
)

func UploadCheckpointByDA(checkpoint *checkpoint.Checkpoint, pk, gasCoupon, namespaceID, network string, timeout time.Duration) error {
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
		sdk.WithGasCode(gasCoupon),
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

func UploadCheckpointByS3(c *checkpoint.Checkpoint, accessKey, secretKey, region, bucket string, timeout time.Duration) error {
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
