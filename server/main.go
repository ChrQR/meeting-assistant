package main

import (
	"context"
	"log"
	"os"

	"github.com/ChrQR/meeting-assistant/server/internal/client"
	"github.com/ChrQR/meeting-assistant/server/internal/repository"
	"github.com/ChrQR/meeting-assistant/server/internal/server"
	"github.com/ChrQR/meeting-assistant/server/internal/service"
)

//go:generate go tool oapi-codegen -config internal/api/config.yaml internal/api/api.yaml
func main() {
	ctx := context.Background()

	bucketName := os.Getenv("AWS_BUCKET_NAME")
	if bucketName == "" {
		log.Fatal("AWS_BUCKET_NAME not set")
	}
	openrouterAPIKey := os.Getenv("OPENROUTER_API_KEY")
	if openrouterAPIKey == "" {
		log.Fatal("OPENROUTER_API_KEY not set")
	}

	storageClient, err := client.NewStorageClient(ctx, bucketName)
	if err != nil {
		log.Fatalf("Unable to initiate storage client: %v", err)
	}

	dbRepository := repository.NewDBRepository(db database.Queries)
	transciptionClient := client.NewTranscriptionClient(openrouterAPIKey)

	transcriptionService := service.NewTranscriptionService(transciptionClient, storageClient, dbRepository)

	server := server.NewServer(transcriptionService)
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
