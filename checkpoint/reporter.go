package checkpoint

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/RiemaLabs/indexer-committee/ord"
)

func NewCheckpoint(indexID IndexerIdentification, header ord.Header) Checkpoint {
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

const (
	// AWS_S3_REGION = "AWS_REGION"
	AWS_S3_REGION = "us-west-2"
	AWS_S3_BUCKET = "nubit-modular-indexer"
)

// URI s3://arn:aws:s3:us-west-2:905418332373:accesspoint/ap-indexer
// We will be using this client everywhere in our code
var awsS3Client *s3.Client

// OBS: "config" is a key word in "github.com/aws/aws-sdk-go-v2/config", cannot be used as argument name here
func UploadCheckpoint(history UploadHistory, indexerID IndexerIdentification, checkpoint Checkpoint) {
	// the SDK uses its default credential chain to find AWS credentials. This default credential chain looks for credentials in the following order:aws.Configconfig.LoadDefaultConfig
	// creds := credentials.NewStaticCredentialsProvider(your_access_key, your_secret_key, "")
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(AWS_S3_REGION))
	// 	config.WithSharedCredentialsFiles(
	// 		[]string{"test/credentials", "data/credentials"},
	// 	),
	// 	config.WithSharedConfigFiles(
	// 		[]string{"test/config", "data/config"},
	// 	)

	if err != nil {
		log.Fatal(err)
	}

	awsS3Client = s3.NewFromConfig(cfg)
	uploader := manager.NewUploader(awsS3Client)
	// downloader := manager.NewDownloader(awsS3Client)

	mac, err := getMACAddress()
	if err != nil {
		log.Fatal(err)
	}
	objectKey := fmt.Sprintf("test/checkpoint-%s-%s-%s-%s.json",
		mac, checkpoint.MetaProtocol, checkpoint.Height, checkpoint.Hash)

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
		Bucket: aws.String(AWS_S3_BUCKET),
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

func getMACAddress() (string, error) {
	// all interfaces info
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	// the first MAC addr of non-vertical interface
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
			// filter virtual and loop interfaces
			// println(iface.HardwareAddr.String())
			return iface.HardwareAddr.String(), nil
		}
	}

	return "", fmt.Errorf("no active non-loopback network interface found")
}
