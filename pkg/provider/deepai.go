package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

const defaultDeepAIBaseURL = "https://api.deepai.org"

type DeepAIProvider struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

func NewDeepAIProvider(apiKey, baseURL string) *DeepAIProvider {
	if baseURL == "" {
		baseURL = defaultDeepAIBaseURL
	}
	baseURL = trimSlash(baseURL)
	return &DeepAIProvider{
		APIKey:  apiKey,
		BaseURL: baseURL,
		HTTP: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (p *DeepAIProvider) Name() string { return "deepai" }

type deepAIResponse struct {
	ID        string `json:"id"`
	OutputURL string `json:"output_url"`
}

func (p *DeepAIProvider) GenerateImage(params *GenerateParams) (*GenerateResult, error) {
	endpoint := p.BaseURL + "/api/text2img"

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("text", params.Prompt)

	if params.Size != "" {
		w, h := parseSize(params.Size)
		if w > 0 {
			_ = writer.WriteField("width", fmt.Sprintf("%d", w))
		}
		if h > 0 {
			_ = writer.WriteField("height", fmt.Sprintf("%d", h))
		}
	}
	if params.Quality != "" {
		_ = writer.WriteField("image_generator_version", params.Quality)
	}
	if params.Style != "" {
		_ = writer.WriteField("genius_preference", params.Style)
	}
	_ = writer.Close()

	req, err := http.NewRequest("POST", endpoint, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("api-key", p.APIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := p.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DeepAI API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result deepAIResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if result.OutputURL == "" {
		return nil, fmt.Errorf("no output URL in response")
	}

	imageData, err := downloadImage(p.HTTP, result.OutputURL)
	if err != nil {
		return nil, fmt.Errorf("downloading image: %w", err)
	}

	return &GenerateResult{Data: imageData}, nil
}
