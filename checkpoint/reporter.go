package checkpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	sdk "github.com/RiemaLabs/nubit-da-sdk"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
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

func UploadCheckpointByS3(indexerID *IndexerIdentification, c *Checkpoint, bucket, objectKey string, cfg *aws.Config, timeout time.Duration) error {
	var awsS3Client = s3.NewFromConfig(*cfg)
	uploader := manager.NewUploader(awsS3Client)

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

func DownloadCheckpointByS3(indexerID IndexerIdentification, writer *io.WriterAt, region, bucket, objectKey string, timeout time.Duration) error {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return err
	}

	var awsS3Client = s3.NewFromConfig(cfg)
	downloader := manager.NewDownloader(awsS3Client)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel() // release resources if the operation completes before the timeout elapses

	numBytes, err := downloader.Download(ctx, *writer, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		log.Printf("Failed to download file, error: %v\n", err)
		return err
	}
	log.Printf("File with size %d downloaded successfully!\n", numBytes)

	return nil
}

func UploadCheckpointByDA(indexerID *IndexerIdentification, checkpoint *Checkpoint, daRPC, pk, inviteCode, namespaceID, network string, timeout time.Duration) error {
	checkpointJSON, err := json.Marshal(checkpoint)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	sdk.SetNet(network)
	clientDA := sdk.NewNubit(sdk.WithCtx(ctx),
		sdk.WithRpc(daRPC),
		sdk.WithInviteCode(inviteCode),
		sdk.WithPrivateKey(pk),
	)
	if clientDA == nil {
		return fmt.Errorf("cannot build the Nubit client")
	}

	labels := map[string]interface{}{
		"contentType": "application/json",
	}
	_, err = clientDA.UploadBytes(checkpointJSON, namespaceID, 0, labels)
	if err != nil {
		log.Println("Failed to upload checkpoint:", err)
		return err
	}
	return nil
}
