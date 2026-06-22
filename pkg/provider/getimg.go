package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultGetImgBaseURL = "https://api.getimg.ai/v2"

type GetImgProvider struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

func NewGetImgProvider(apiKey, baseURL string) *GetImgProvider {
	if baseURL == "" {
		baseURL = defaultGetImgBaseURL
	}
	baseURL = trimSlash(baseURL)
	return &GetImgProvider{
		APIKey:  apiKey,
		BaseURL: baseURL,
		HTTP: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (p *GetImgProvider) Name() string { return "getimg" }

type getImgRequest struct {
	Model       string `json:"model"`
	Prompt      string `json:"prompt"`
	AspectRatio string `json:"aspect_ratio,omitempty"`
	Resolution  string `json:"resolution,omitempty"`
	OutputFmt   string `json:"output_format,omitempty"`
}

type getImgResponse struct {
	Status string         `json:"status"`
	Data   []getImgData   `json:"data"`
	Usage  getImgUsage    `json:"usage"`
}

type getImgData struct {
	URL     string `json:"url"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
}

type getImgUsage struct {
	TotalCost float64 `json:"total_cost"`
}

func (p *GetImgProvider) GenerateImage(params *GenerateParams) (*GenerateResult, error) {
	model := params.Model
	if model == "" {
		model = "seedream-5-lite"
	}

	reqBody := getImgRequest{
		Model:       model,
		Prompt:      params.Prompt,
		AspectRatio: sizeToGetImgAR(params.Size),
		Resolution:  qualityToResolution(params.Quality),
		OutputFmt:   "png",
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
		return nil, fmt.Errorf("GetImg API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result getImgResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if result.Status != "completed" {
		return nil, fmt.Errorf("generation not completed (status: %s)", result.Status)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no images returned")
	}

	imageData, err := downloadImage(p.HTTP, result.Data[0].URL)
	if err != nil {
		return nil, fmt.Errorf("downloading image: %w", err)
	}

	return &GenerateResult{Data: imageData}, nil
}

func sizeToGetImgAR(size string) string {
	switch size {
	case "1024x1024":
		return "1:1"
	case "1280x720", "1920x1080":
		return "16:9"
	case "720x1280", "1080x1920":
		return "9:16"
	case "1152x896":
		return "9:7"
	case "1216x832":
		return "3:2"
	case "1344x768":
		return "16:9"
	case "1536x640":
		return "21:9"
	case "896x1152":
		return "7:9"
	case "832x1216":
		return "2:3"
	case "768x1344":
		return "9:16"
	case "640x1536":
		return "9:21"
	default:
		return "1:1"
	}
}

func qualityToResolution(quality string) string {
	switch quality {
	case "hd", "2k", "2K":
		return "2K"
	case "4k", "4K", "ultra":
		return "4K"
	default:
		return "1K"
	}
}
