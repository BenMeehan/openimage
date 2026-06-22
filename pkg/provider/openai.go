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

const defaultOpenAIBaseURL = "https://api.openai.com/v1"
const defaultOpenRouterBaseURL = "https://openrouter.ai/api/v1"

type OpenAIProvider struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

func NewOpenAIProvider(apiKey, baseURL string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = defaultOpenRouterBaseURL
	}
	baseURL = trimSlash(baseURL)
	return &OpenAIProvider{
		APIKey:  apiKey,
		BaseURL: baseURL,
		HTTP: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (p *OpenAIProvider) Name() string { return "openai" }

type openAIImageRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	N              int    `json:"n"`
	Size           string `json:"size,omitempty"`
	Quality        string `json:"quality,omitempty"`
	Style          string `json:"style,omitempty"`
	ResponseFormat string `json:"response_format"`
}

type openAIImageResponse struct {
	Data []openAIImageData `json:"data"`
}

type openAIImageData struct {
	B64JSON       string `json:"b64_json"`
	URL           string `json:"url"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type openAIError struct {
	Error struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}

func (p *OpenAIProvider) GenerateImage(params *GenerateParams) (*GenerateResult, error) {
	req := &openAIImageRequest{
		Model:          params.Model,
		Prompt:         params.Prompt,
		N:              params.N,
		Size:           params.Size,
		Quality:        params.Quality,
		Style:          params.Style,
		ResponseFormat: "b64_json",
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", p.BaseURL+"/images/generations", bytes.NewReader(body))
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
		var apiErr openAIError
		if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.Error.Message != "" {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, apiErr.Error.Message)
		}
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var imgResp openAIImageResponse
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

func trimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
