// Package llm provides an optional OpenAI-compatible chat completion client
// used strictly for prose (summaries, explanations, architectural
// descriptions, naming) — never for extracting structural information that
// deterministic parsing already provides (PRD sections 3 & 18).
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/anushibinj/repo-mapper/internal/config"
)

// Client talks to any OpenAI-compatible chat completions endpoint.
type Client struct {
	endpoint   string
	apiKey     string
	model      string
	httpClient *http.Client
}

// New builds a Client from LLM configuration. Returns nil, false if LLM
// integration is disabled or misconfigured — callers should treat that as
// "skip summarisation", never as a fatal error.
func New(cfg config.LLMConfig) (*Client, bool) {
	if !cfg.Enabled || cfg.Endpoint == "" {
		return nil, false
	}
	return &Client{
		endpoint:   cfg.Endpoint,
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, true
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Summarize sends a single prompt and returns the model's text response.
// Never called by deterministic parsing/analysis paths — only by optional
// documentation-enrichment steps.
func (c *Client) Summarize(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var parsed chatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("llm: invalid response: %w", err)
	}
	if parsed.Error != nil {
		return "", errors.New("llm: " + parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", errors.New("llm: empty response")
	}
	return parsed.Choices[0].Message.Content, nil
}
