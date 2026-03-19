package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// EmbeddingProvider generates vector embeddings from text.
type EmbeddingProvider interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// OpenAIEmbedder generates embeddings using OpenAI's API.
type OpenAIEmbedder struct {
	apiKey     string
	model      string
	httpClient *http.Client
	baseURL    string
}

// OpenAIEmbeddingResponse represents the OpenAI API response.
type OpenAIEmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// OpenAIEmbeddingRequest represents the OpenAI API request.
type OpenAIEmbeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

// NewOpenAIEmbedder creates a new OpenAI embedding provider.
func NewOpenAIEmbedder(apiKey string) *OpenAIEmbedder {
	return &OpenAIEmbedder{
		apiKey:     apiKey,
		model:      "text-embedding-3-small",
		httpClient: &http.Client{},
		baseURL:    "https://api.openai.com/v1",
	}
}

// NewOpenAIEmbedderFromEnv creates a new OpenAI embedder using OPENAI_API_KEY env var.
func NewOpenAIEmbedderFromEnv() *OpenAIEmbedder {
	return NewOpenAIEmbedder(os.Getenv("OPENAI_API_KEY"))
}

// Embed generates a vector embedding for the given text.
func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if e.apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key not configured")
	}

	reqBody := OpenAIEmbeddingRequest{
		Model: e.model,
		Input: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API returned status %d", resp.StatusCode)
	}

	var embedResp OpenAIEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(embedResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	return embedResp.Data[0].Embedding, nil
}

// SetHTTPClient allows customizing the HTTP client (for testing).
func (e *OpenAIEmbedder) SetHTTPClient(client *http.Client) {
	e.httpClient = client
}
