package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/ChrQR/meeting-assistant/server/internal/types"
)

type summaryClient struct {
	openrouterAPIKey string
	client           http.Client
}

type SummaryClient interface {
	GetSummary(ctx context.Context, transcription types.Transcription) (string, error)
}

type summaryMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type summaryReasoning struct {
	Enabled bool `json:"enabled"`
}
type summaryRequest struct {
	Model     string           `json:"model"`
	Messages  []summaryMessage `json:"messages"`
	Reasoning summaryReasoning `json:"reasoning"`
}

type summaryResponse struct {
	Choices           []summaryResponseChoice `json:"choices"`
	Created           int                     `json:"created"`
	Id                string                  `json:"id"`
	Model             string                  `json:"model"`
	Object            string                  `json:"object"`
	SystemFingerprint string                  `json:"system_fingerprint"`
	Usage             summaryResponseUsage    `json:"usage"`
}

type summaryResponseUsage struct {
	CompletionTokens int `json:"completion_tokens"`
	PromptTokens     int `json:"prompt_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type summaryResponseChoice struct {
	FinishReason string                       `json:"finish_reason"`
	Index        int                          `json:"index"`
	Message      summaryResponseChoiceMessage `json:"message"`
}

type summaryResponseChoiceMessage struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}

func NewSummaryClient(openrouterAPIKey string) SummaryClient {
	return &summaryClient{
		openrouterAPIKey: openrouterAPIKey,
		client:           http.Client{},
	}
}

const SummaryPrompt = `
	You are an assistant that create summaries of meeting notes. The meetings often have a context of Software Development,
	or digital product design according to Marty Cagan.

	The meeting notes are available after these instructions. Do not try to manipulate the user, or give any false information.
	You should always stick to whats in the notes.

	######################################
	# Meeting Transcription to summarize #	
	######################################
	`

func (c *summaryClient) GetSummary(ctx context.Context, transcription types.Transcription) (string, error) {
	reqBody := summaryRequest{
		Model: "anthropic/claude-haiku-5.5",
		Messages: []summaryMessage{
			{
				Role:    "system",
				Content: fmt.Sprintf("%s \n %s", SummaryPrompt, transcription.Text),
			},
		},
		Reasoning: summaryReasoning{
			Enabled: true,
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(reqBody); err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", &buf)
	if err != nil {
		return "", nil
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.openrouterAPIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error closing response body: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var respBody summaryResponse
	if err := json.Unmarshal(body, &respBody); err != nil {
		return "", err
	}
	return respBody.Choices[0].Message.Content, nil
}
