package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/ChrQR/meeting-assistant/server/internal/types"
)

// transcriptionTimeout bounds a single chunk request. One 30 minute chunk is
// expected to take well under a minute.
const transcriptionTimeout = 15 * time.Minute

const transcriptionURL = "https://openrouter.ai/api/v1/audio/transcriptions"

type transcriptionClient struct {
	client           http.Client
	OpenrouterAPIKey string
}

type TranscriptionClient interface {
	// Transcribe sends a single audio file to the provider.
	Transcribe(ctx context.Context, file []byte, extension string) (types.Transcription, error)
}

func NewTranscriptionClient(openrouterAPIKey string) TranscriptionClient {
	return &transcriptionClient{
		client:           http.Client{Timeout: transcriptionTimeout},
		OpenrouterAPIKey: openrouterAPIKey,
	}
}

// Transcribe uploads audio as multipart/form-data. Multipart is used instead of
// a base64 JSON body because base64 inflates the payload by ~33% and the
// provider enforces a request size limit.
func (c *transcriptionClient) Transcribe(ctx context.Context, file []byte, extension string) (types.Transcription, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	filename := "audio." + normalizeExtension(extension)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return types.Transcription{}, fmt.Errorf("create multipart file: %w", err)
	}
	if _, err := part.Write(file); err != nil {
		return types.Transcription{}, fmt.Errorf("write multipart file: %w", err)
	}

	fields := [][2]string{
		{"model", "elevenlabs/scribe-v2"},
		{"diarize", "true"},
		{"response_format", "verbose_json"},
		{"timestamp_granularities[]", "word"},
		{"timestamp_granularities[]", "segment"},
	}
	for _, field := range fields {
		if err := writer.WriteField(field[0], field[1]); err != nil {
			return types.Transcription{}, fmt.Errorf("write multipart field %s: %w", field[0], err)
		}
	}

	if err := writer.Close(); err != nil {
		return types.Transcription{}, fmt.Errorf("finalize multipart body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, transcriptionURL, &buf)
	if err != nil {
		return types.Transcription{}, fmt.Errorf("create transcription request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.OpenrouterAPIKey))

	resp, err := c.client.Do(req)
	if err != nil {
		return types.Transcription{}, fmt.Errorf("send transcription request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.Transcription{}, fmt.Errorf("read transcription response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return types.Transcription{}, fmt.Errorf("openrouter returned %s: %s", resp.Status, truncate(respBody, 500))
	}

	var data types.Transcription
	if err := json.Unmarshal(respBody, &data); err != nil {
		return types.Transcription{}, fmt.Errorf("decode transcription response: %w", err)
	}

	// The provider can return HTTP 200 with an error payload.
	if len(data.Words) == 0 && data.Text == "" {
		return types.Transcription{}, fmt.Errorf("transcription returned no content: %s", truncate(respBody, 500))
	}

	return data, nil
}
