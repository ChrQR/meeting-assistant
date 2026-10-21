package main

import (
	"context"
	"log"
	"os"
	"strings"

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

	azureClientID := requireEnv("AZURE_CLIENT_ID")
	azureTenantID := requireEnv("AZURE_TENANT_ID")
	azureClientSecret := requireEnv("AZURE_CLIENT_SECRET")
	authSecret := requireEnv("AUTH_SECRET")
	appURL := requireEnv("APP_URL")
	oauthCallbackURL := os.Getenv("OAUTH_CALLBACK_URL")
	if oauthCallbackURL == "" {
		oauthCallbackURL = "http://localhost:8080/auth/callback"
	}

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
	mediaClient := client.NewMediaClient()

	if !mediaClient.Available() {
		log.Fatal("ffmpeg and ffprobe must be installed and on PATH")
	}

	transcriptionService := service.NewTranscriptionService(
		transciptionClient,
		storageClient,
		embeddingClient,
		summaryClient,
		mediaClient,
		dbRepository,
	)

	storeService := service.NewStoreService(dbRepository, storageClient)

	providers := client.NewProviderRegistry(
		client.NewMicrosoftProvider(azureTenantID, azureClientID, azureClientSecret, oauthCallbackURL),
	)
	authService := service.NewAuthService(dbRepository, providers)
	signer := client.NewSessionSigner(authSecret, client.SessionTTL)

	server := server.NewServer(server.Options{
		TranscriptionService: transcriptionService,
		StoreService:         storeService,
		AuthService:          authService,
		Signer:               signer,
		AppURL:               appURL,
		SecureCookies:        strings.HasPrefix(appURL, "https://"),
	})
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}

func requireEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%s not set", key)
	}
	return value
}
