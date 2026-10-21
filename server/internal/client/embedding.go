package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ChrQR/meeting-assistant/server/internal/types"
)

const embeddingTimeout = 5 * time.Minute

type embeddingClient struct {
	URL    string
	client http.Client
}

type EmbeddingClient interface {
	EmbedSpeakers(ctx context.Context, recording []byte, speakers []types.SpeakerData) (types.EmbeddingResponse, error)
}

func NewEmbeddingClient(url string) EmbeddingClient {
	return &embeddingClient{
		URL:    url,
		client: http.Client{Timeout: embeddingTimeout},
	}
}

func (c *embeddingClient) EmbedSpeakers(ctx context.Context, recording []byte, speakers []types.SpeakerData) (types.EmbeddingResponse, error) {
	if len(speakers) == 0 {
		return types.EmbeddingResponse{Data: []types.EmbeddingResult{}}, nil
	}

	marshalledData, err := json.Marshal(types.EmbeddingRequest{
		File:     recording,
		Speakers: speakers,
	})
	if err != nil {
		return types.EmbeddingResponse{}, fmt.Errorf("marshal embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(marshalledData))
	if err != nil {
		return types.EmbeddingResponse{}, fmt.Errorf("create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return types.EmbeddingResponse{}, fmt.Errorf("send embedding request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.EmbeddingResponse{}, fmt.Errorf("read embedding response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return types.EmbeddingResponse{}, fmt.Errorf("embedding service returned %s: %s", resp.Status, truncate(body, 500))
	}

	var respData types.EmbeddingResponse
	if err := json.Unmarshal(body, &respData); err != nil {
		return types.EmbeddingResponse{}, fmt.Errorf("decode embedding response: %w", err)
	}

	return respData, nil
}

// truncate keeps error messages readable by limiting how much upstream body is included.
func truncate(body []byte, limit int) string {
	if len(body) <= limit {
		return string(body)
	}
	return string(body[:limit]) + "…"
}
