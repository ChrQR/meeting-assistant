package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type storageClient struct {
	client     *s3.Client
	bucketName string
}

type StorageClient interface {
	InsertRecording(ctx context.Context, fileBytes []byte, userPath string, fileName string, extension string) error
	GetRecording(ctx context.Context, userPath string, fileName string, extension string) ([]byte, error)
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
		Key:         new(recordingKey(userPath, fileName, extension)),
		Body:        bytes.NewReader(fileBytes),
		ContentType: &contentType,
	})
	if err != nil {
		return err
	}
	return nil
}

// recordingKey builds the object key. The uploaded filename usually already
// carries an extension, so a duplicate is stripped before appending the real
// one (otherwise "meeting.mp4" would be stored as "meeting.mp4.mp4").
func recordingKey(userPath string, fileName string, extension string) string {
	base := strings.TrimSuffix(filepath.Base(fileName), filepath.Ext(fileName))
	if base == "" {
		base = "recording"
	}
	return fmt.Sprintf("%s/%s.%s", userPath, base, strings.TrimPrefix(extension, "."))
}

func (c *storageClient) GetRecording(ctx context.Context, userPath string, fileName string, extension string) ([]byte, error) {
	output, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &c.bucketName,
		Key:    new(recordingKey(userPath, fileName, extension)),
	})
	if err != nil {
		return nil, err
	}
	defer output.Body.Close()

	return io.ReadAll(output.Body)
}
