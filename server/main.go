package main

import (
	"context"
	"log"
	"os"

	"github.com/ChrQR/meeting-assistant/server/internal/client"
	"github.com/ChrQR/meeting-assistant/server/internal/database"
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

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	embeddingServiceURL := os.Getenv("EMBEDDING_SERVICE_URL")

	storageClient, err := client.NewStorageClient(ctx, bucketName)
	if err != nil {
		log.Fatalf("Unable to initiate storage client: %v", err)
	}

	db, err := database.NewConnection(ctx, dbURL)
	if err != nil {
		log.Fatalf("Error initiating db connection: %v", err)
	}

	dbRepository := repository.NewDBRepository(db)
	transciptionClient := client.NewTranscriptionClient(openrouterAPIKey)
	embeddingClient := client.NewEmbeddingClient(embeddingServiceURL)
	summaryClient := client.NewSummaryClient(openrouterAPIKey)

	transcriptionService := service.NewTranscriptionService(
		transciptionClient,
		storageClient,
		embeddingClient,
		summaryClient,
		dbRepository,
	)

	server := server.NewServer(transcriptionService)
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
