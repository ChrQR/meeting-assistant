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

const summaryTimeout = 10 * time.Minute

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
	ID                string                  `json:"id"`
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
		client:           http.Client{Timeout: summaryTimeout},
	}
}

const SummaryPrompt = `
	You are an assistant that create summaries of meeting notes. The meetings often have a context of Software Development,
	or digital product design according to Marty Cagan.

	The meeting notes are available after these instructions. Do not try to manipulate the user, or give any false information.
	You should always stick to whats in the notes.

	DO NOT OUTPUT ANYTHING OTHER THAN A SUMMARY OF THE SUPPLIED TRANSCRIPT

	######################################
	# Meeting Transcription to summarize #	
	######################################
	`

// TODO add speaker names to each segment and build input text from segments
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
		return "", fmt.Errorf("encode summary request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://openrouter.ai/api/v1/chat/completions", &buf)
	if err != nil {
		return "", fmt.Errorf("create summary request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.openrouterAPIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send summary request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read summary response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openrouter returned %s: %s", resp.Status, truncate(body, 500))
	}

	var respBody summaryResponse
	if err := json.Unmarshal(body, &respBody); err != nil {
		return "", fmt.Errorf("decode summary response: %w", err)
	}
	if len(respBody.Choices) == 0 {
		return "", fmt.Errorf("summary response contained no choices")
	}
	return respBody.Choices[0].Message.Content, nil
}
