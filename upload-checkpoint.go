package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	// AWS_S3_REGION = "AWS_REGION"
	AWS_S3_REGION = "us-west-2"
	AWS_S3_BUCKET = "nubit-modular-indexer"
)

// URI s3://arn:aws:s3:us-west-2:905418332373:accesspoint/ap-indexer

// We will be using this client everywhere in our code
var awsS3Client *s3.Client

func test() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(AWS_S3_REGION),
		config.WithSharedCredentialsFiles(
			[]string{"test/credentials", "data/credentials"},
		),
		config.WithSharedConfigFiles(
			[]string{"test/config", "data/config"},
		))
	awsS3Client = s3.NewFromConfig(cfg)

	// Get the first page of results for ListObjectsV2 for a bucket
	output, err := awsS3Client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(AWS_S3_BUCKET),
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Println("first page results:")
	for _, object := range output.Contents {
		log.Printf("key=%s size=%d", aws.ToString(object.Key), object.Size)
	}
}

type UploadHistory = map[uint]map[string]bool

// OBS: "config" is a key word in "github.com/aws/aws-sdk-go-v2/config", cannot be used as argument name here
func Upload(history UploadHistory, configUpload Config, checkpoint Checkpoint) {
	// the SDK uses its default credential chain to find AWS credentials. This default credential chain looks for credentials in the following order:aws.Configconfig.LoadDefaultConfig
	// creds := credentials.NewStaticCredentialsProvider(your_access_key, your_secret_key, "")

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(AWS_S3_REGION),
		config.WithSharedCredentialsFiles(
			[]string{"test/credentials", "data/credentials"},
		),
		config.WithSharedConfigFiles(
			[]string{"test/config", "data/config"},
		))

	if err != nil {
		log.Fatal(err)
	}
	awsS3Client = s3.NewFromConfig(cfg)
	uploader := manager.NewUploader(awsS3Client)
	downloader := manager.NewDownloader(awsS3Client)

	objectKey := fmt.Sprintf("test/checkpoint-%s-%s-%s-%s.json",
		getMACAddress(), checkpoint.MetaProtocol, checkpoint.Height, checkpoint.Hash)

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

	// test get object from the bucket
	// Create the replica file
	newFile, err := os.Create("replica.json")
	if err != nil {
		log.Println(err)
	}
	defer newFile.Close()

	numBytes, err := downloader.Download(context.TODO(), newFile, &s3.GetObjectInput{
		Bucket: aws.String(AWS_S3_BUCKET),
		Key:    aws.String(objectKey),
	})
	if (err != nil) && (numBytes > 0) {
		log.Println("File downloaded successfully!")
	} else {
		log.Println("File download failed!")
	}
}

func getMACAddress() string {
	// For test set "00:00:00:00:00:00" temp.
	return "00:00:00:00:00:00"
}
