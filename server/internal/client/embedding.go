package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/ChrQR/meeting-assistant/server/internal/types"
)

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
		client: http.Client{},
	}
}

func (c *embeddingClient) EmbedSpeakers(ctx context.Context, recording []byte, speakers []types.SpeakerData) (types.EmbeddingResponse, error) {
	reqData := types.EmbeddingRequest{
		File:     recording,
		Speakers: speakers,
	}
	marshalledData, err := json.Marshal(reqData)
	if err != nil {
		log.Printf("Error marshalling embedding request: %v", err)
		return types.EmbeddingResponse{}, err
	}
	req, err := http.NewRequest("POST", c.URL, bytes.NewReader(marshalledData))
	if err != nil {
		log.Printf("Error creating embedding request: %v", err)
		return types.EmbeddingResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		log.Printf("Error sending request: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Unable to close response body: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.EmbeddingResponse{}, err
	}
	var respData types.EmbeddingResponse
	if err := json.Unmarshal(body, &respData); err != nil {
		return types.EmbeddingResponse{}, err
	}

	return respData, nil
}
