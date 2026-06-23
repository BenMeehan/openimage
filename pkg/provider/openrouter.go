package provider

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultOpenRouterBaseURL = "https://openrouter.ai/api/v1"

type OpenRouterProvider struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

func NewOpenRouterProvider(apiKey, baseURL string) *OpenRouterProvider {
	if baseURL == "" {
		baseURL = defaultOpenRouterBaseURL
	}
	baseURL = trimSlash(baseURL)
	return &OpenRouterProvider{
		APIKey:  apiKey,
		BaseURL: baseURL,
		HTTP: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (p *OpenRouterProvider) Name() string { return "openrouter" }

type openRouterImageRequest struct {
	Model          string                `json:"model"`
	Prompt         string                `json:"prompt"`
	N              int                   `json:"n,omitempty"`
	Size           string                `json:"size,omitempty"`
	Resolution     string                `json:"resolution,omitempty"`
	AspectRatio    string                `json:"aspect_ratio,omitempty"`
	Quality        string                `json:"quality,omitempty"`
	OutputFormat   string                `json:"output_format,omitempty"`
	Background     string                `json:"background,omitempty"`
	OutputCompress int                   `json:"output_compression,omitempty"`
	References     []openRouterReference `json:"input_references,omitempty"`
}

type openRouterReference struct {
	Type     string                 `json:"type"`
	ImageURL *openRouterImageURL    `json:"image_url"`
}

type openRouterImageURL struct {
	URL string `json:"url"`
}

type openRouterImageResponse struct {
	Data  []openRouterImageData `json:"data"`
	Usage *openRouterUsage      `json:"usage,omitempty"`
}

type openRouterImageData struct {
	B64JSON       string `json:"b64_json"`
	URL           string `json:"url,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type openRouterUsage struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	Cost             float64 `json:"cost"`
}

func (p *OpenRouterProvider) GenerateImage(params *GenerateParams) (*GenerateResult, error) {
	req := &openRouterImageRequest{
		Model:  params.Model,
		Prompt: params.Prompt,
	}

	if params.N > 1 {
		req.N = params.N
	}
	if params.Size != "" {
		req.Size = params.Size
	}
	if params.Quality != "" {
		req.Quality = params.Quality
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", p.BaseURL+"/images", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	httpReq.Header.Set("User-Agent", "openimage/1.0")

	resp, err := p.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var imgResp openRouterImageResponse
	if err := json.Unmarshal(respBody, &imgResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if len(imgResp.Data) == 0 {
		return nil, fmt.Errorf("no images returned")
	}

	var imageData []byte
	if imgResp.Data[0].B64JSON != "" {
		imageData, err = base64.StdEncoding.DecodeString(imgResp.Data[0].B64JSON)
		if err != nil {
			return nil, fmt.Errorf("decoding image: %w", err)
		}
	} else if imgResp.Data[0].URL != "" {
		imageData, err = downloadImage(p.HTTP, imgResp.Data[0].URL)
		if err != nil {
			return nil, fmt.Errorf("downloading image: %w", err)
		}
	} else {
		return nil, fmt.Errorf("no image data in response")
	}

	return &GenerateResult{
		Data:          imageData,
		RevisedPrompt: imgResp.Data[0].RevisedPrompt,
	}, nil
}
