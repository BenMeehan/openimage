package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

const defaultIdeogramBaseURL = "https://api.ideogram.ai/v1"

type IdeogramProvider struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

func NewIdeogramProvider(apiKey, baseURL string) *IdeogramProvider {
	if baseURL == "" {
		baseURL = defaultIdeogramBaseURL
	}
	baseURL = trimSlash(baseURL)
	return &IdeogramProvider{
		APIKey:  apiKey,
		BaseURL: baseURL,
		HTTP: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (p *IdeogramProvider) Name() string { return "ideogram" }

type ideogramResponse struct {
	Data []ideogramData `json:"data"`
}

type ideogramData struct {
	URL        string `json:"url"`
	Resolution string `json:"resolution"`
	Seed       int64  `json:"seed"`
}

func (p *IdeogramProvider) GenerateImage(params *GenerateParams) (*GenerateResult, error) {
	model := strings.ToLower(params.Model)
	version := "ideogram-v3"
	if strings.Contains(model, "v4") || strings.Contains(model, "v-4") || strings.Contains(model, "ideogram-4") {
		version = "ideogram-v4"
	} else if strings.Contains(model, "v2") || strings.Contains(model, "v-2") {
		version = "ideogram-v2"
	} else if strings.Contains(model, "v3") || strings.Contains(model, "v-3") {
		version = "ideogram-v3"
	}

	endpoint := fmt.Sprintf("%s/%s/generate", p.BaseURL, version)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("prompt", params.Prompt)
	_ = writer.WriteField("aspct_ratio", sizeToIdeogramAR(params.Size))

	if params.Style != "" {
		_ = writer.WriteField("style_type", strings.ToUpper(params.Style))
	}
	if params.Quality != "" {
		_ = writer.WriteField("rendering_speed", strings.ToUpper(params.Quality))
	}
	if params.N > 1 {
		_ = writer.WriteField("num_images", fmt.Sprintf("%d", params.N))
	}
	_ = writer.Close()

	req, err := http.NewRequest("POST", endpoint, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Api-Key", p.APIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := p.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ideogram API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result ideogramResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
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

func sizeToIdeogramAR(size string) string {
	switch size {
	case "1024x1024":
		return "1x1"
	case "1280x720", "1920x1080":
		return "16x9"
	case "720x1280", "1080x1920":
		return "9x16"
	case "1152x896", "1344x768":
		return "3x2"
	case "896x1152", "768x1344":
		return "2x3"
	default:
		return "1x1"
	}
}


