package provider

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

const defaultClipdropBaseURL = "https://clipdrop-api.co"

type ClipdropProvider struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

func NewClipdropProvider(apiKey, baseURL string) *ClipdropProvider {
	if baseURL == "" {
		baseURL = defaultClipdropBaseURL
	}
	baseURL = trimSlash(baseURL)
	return &ClipdropProvider{
		APIKey:  apiKey,
		BaseURL: baseURL,
		HTTP: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (p *ClipdropProvider) Name() string { return "clipdrop" }

func (p *ClipdropProvider) GenerateImage(params *GenerateParams) (*GenerateResult, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("prompt", params.Prompt)
	_ = writer.Close()

	req, err := http.NewRequest("POST", p.BaseURL+"/text-to-image/v1", &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := p.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Clipdrop API error (status %d): %s", resp.StatusCode, string(body))
	}

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return &GenerateResult{Data: imageData}, nil
}
