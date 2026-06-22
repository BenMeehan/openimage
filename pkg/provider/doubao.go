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

const defaultDoubaoBaseURL = "https://ark.cn-beijing.volces.com/api/v3"

type DoubaoProvider struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

func NewDoubaoProvider(apiKey, baseURL string) *DoubaoProvider {
	if baseURL == "" {
		baseURL = defaultDoubaoBaseURL
	}
	baseURL = trimSlash(baseURL)
	return &DoubaoProvider{
		APIKey:  apiKey,
		BaseURL: baseURL,
		HTTP: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (p *DoubaoProvider) Name() string { return "doubao" }

type doubaoImageRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	N              int    `json:"n"`
	Size           string `json:"size,omitempty"`
	ResponseFormat string `json:"response_format"`
}

type doubaoImageResponse struct {
	Data []doubaoImageData `json:"data"`
}

type doubaoImageData struct {
	URL     string `json:"url"`
	B64JSON string `json:"b64_json"`
}

func (p *DoubaoProvider) GenerateImage(params *GenerateParams) (*GenerateResult, error) {
	model := params.Model
	if model == "" {
		model = "doubao-seedream-3.0-t2i-250415"
	}

	reqBody := doubaoImageRequest{
		Model:          model,
		Prompt:         params.Prompt,
		N:              params.N,
		Size:           params.Size,
		ResponseFormat: "b64_json",
	}

	body, _ := json.Marshal(reqBody)

	httpReq, err := http.NewRequest("POST", p.BaseURL+"/images/generations", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	httpReq.Header.Set("User-Agent", "openimage/1.0")

	resp, err := p.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Doubao API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result doubaoImageResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no images returned")
	}

	var imageData []byte
	if result.Data[0].B64JSON != "" {
		imageData, err = base64.StdEncoding.DecodeString(result.Data[0].B64JSON)
		if err != nil {
			return nil, fmt.Errorf("decoding image: %w", err)
		}
	} else if result.Data[0].URL != "" {
		imageData, err = downloadImage(p.HTTP, result.Data[0].URL)
		if err != nil {
			return nil, fmt.Errorf("downloading image: %w", err)
		}
	} else {
		return nil, fmt.Errorf("no image data in response")
	}

	return &GenerateResult{Data: imageData}, nil
}
