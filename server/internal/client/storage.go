package client

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type storageClient struct {
	client     *s3.Client
	bucketName string
}

type StorageClient interface {
	InsertRecording(ctx context.Context, fileBytes []byte, userPath string, fileName string, extension string) error
}

func NewStorageClient(ctx context.Context, bucketName string) (StorageClient, error) {
	sdkConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Printf("unable to create s3 env config: %v", err)
		return &storageClient{}, err
	}

	client := s3.NewFromConfig(sdkConfig)
	return &storageClient{
		client,
		bucketName,
	}, nil
}

func (c *storageClient) InsertRecording(ctx context.Context, fileBytes []byte, userPath string, fileName string, extension string) error {
	contentType := http.DetectContentType(fileBytes)
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &c.bucketName,
		Key:         new(fmt.Sprintf("%s/%s.%s", userPath, fileName, extension)),
		Body:        bytes.NewReader(fileBytes),
		ContentType: &contentType,
	})
	if err != nil {
		return err
	}
	return nil
}
