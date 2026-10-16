package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/ChrQR/meeting-assistant/server/internal/types"
)

type TranscriptionRequest struct {
	Model                  string    `json:"model"`
	InputAudio             AudioData `json:"input_audio"`
	Diarize                bool      `json:"diarize"`
	ReponseFormat          string    `json:"response_format"`
	TimestampGranularities []string  `json:"timestamp_granularities"`
}

type AudioData struct {
	Data   string `json:"data"`
	Format string `json:"format"`
}

type transcriptionClient struct {
	client           http.Client
	OpenrouterAPIKey string
}

type TranscriptionClient interface {
	Transcribe(file []byte, extension string) (types.Transcription, error)
}

func NewTranscriptionClient(openrouterAPIKey string) TranscriptionClient {
	return &transcriptionClient{
		client:           http.Client{},
		OpenrouterAPIKey: openrouterAPIKey,
	}
}

func (c *transcriptionClient) Transcribe(file []byte, extension string) (types.Transcription, error) {
	body := TranscriptionRequest{
		Model: "elevenlabs/scribe-v2",
		InputAudio: AudioData{
			Data: base64.StdEncoding.EncodeToString(file), Format: extension,
		},
		Diarize:                true,
		ReponseFormat:          "verbose_json",
		TimestampGranularities: []string{"word"},
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return types.Transcription{}, err
	}
	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/audio/transcriptions", &buf)
	if err != nil {
		return types.Transcription{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.OpenrouterAPIKey))

	resp, err := c.client.Do(req)
	if err != nil {
		return types.Transcription{}, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.Transcription{}, err
	}
	if resp.StatusCode != 200 {
		return types.Transcription{}, fmt.Errorf("error from openrouter: %s", resp.Status)
	}

	var data types.Transcription
	if err := json.Unmarshal(respBody, &data); err != nil {
		return types.Transcription{}, err
	}

	data.BuildSegments()
	data.BuildSpeakers()

	return data, nil
}
