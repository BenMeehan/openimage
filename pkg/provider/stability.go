package provider

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

const defaultStabilityBaseURL = "https://api.stability.ai"

type StabilityProvider struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

func NewStabilityProvider(apiKey, baseURL string) *StabilityProvider {
	if baseURL == "" {
		baseURL = defaultStabilityBaseURL
	}
	baseURL = trimSlash(baseURL)
	return &StabilityProvider{
		APIKey:  apiKey,
		BaseURL: baseURL,
		HTTP: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (p *StabilityProvider) Name() string { return "stability" }

type stabilityJSONResponse struct {
	Image       string `json:"image"`
	FinishReason string `json:"finish_reason"`
	Seed        uint32 `json:"seed"`
}

type stabilityV1Response struct {
	Artifacts []stabilityV1Artifact `json:"artifacts"`
}

type stabilityV1Artifact struct {
	Base64       string `json:"base64"`
	FinishReason string `json:"finish_reason"`
	Seed         uint32 `json:"seed"`
}

func (p *StabilityProvider) GenerateImage(params *GenerateParams) (*GenerateResult, error) {
	if p.usesV2Beta(params.Model) {
		return p.generateV2Beta(params)
	}
	return p.generateV1(params)
}

func (p *StabilityProvider) usesV2Beta(model string) bool {
	lower := strings.ToLower(model)
	return strings.Contains(lower, "sd3") ||
		strings.Contains(lower, "sd-3") ||
		strings.Contains(lower, "core") ||
		strings.Contains(lower, "ultra") ||
		strings.Contains(lower, "stable-image")
}

func (p *StabilityProvider) generateV2Beta(params *GenerateParams) (*GenerateResult, error) {
	engine := resolveEngine(params.Model)
	url := fmt.Sprintf("%s/v2beta/stable-image/generate/%s", p.BaseURL, engine)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("prompt", params.Prompt)
	_ = writer.WriteField("output_format", "png")

	if size := params.Size; size != "" {
		_ = writer.WriteField("aspect_ratio", sizeToAspectRatio(size))
	}
	if style := params.Style; style != "" {
		_ = writer.WriteField("style_preset", style)
	}
	if quality := params.Quality; quality != "" {
		_ = writer.WriteField("style_preset", quality)
	}
	_ = writer.Close()

	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := p.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Stability API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result stabilityJSONResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	imageData, err := base64.StdEncoding.DecodeString(result.Image)
	if err != nil {
		return nil, fmt.Errorf("decoding image: %w", err)
	}

	return &GenerateResult{Data: imageData}, nil
}

func (p *StabilityProvider) generateV1(params *GenerateParams) (*GenerateResult, error) {
	engine := resolveEngine(params.Model)
	if engine == "" {
		engine = "stable-diffusion-xl-1024-v1-0"
	}
	url := fmt.Sprintf("%s/v1/generation/%s/text-to-image", p.BaseURL, engine)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("text_prompts[0][text]", params.Prompt)
	_ = writer.WriteField("text_prompts[0][weight]", "1.0")

	if params.Size != "" {
		parts := strings.Split(params.Size, "x")
		if len(parts) == 2 {
			_ = writer.WriteField("width", parts[0])
			_ = writer.WriteField("height", parts[1])
		}
	}
	_ = writer.WriteField("samples", fmt.Sprintf("%d", params.N))
	_ = writer.Close()

	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := p.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Stability API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result stabilityV1Response
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if len(result.Artifacts) == 0 {
		return nil, fmt.Errorf("no images returned")
	}

	imageData, err := base64.StdEncoding.DecodeString(result.Artifacts[0].Base64)
	if err != nil {
		return nil, fmt.Errorf("decoding image: %w", err)
	}

	return &GenerateResult{Data: imageData}, nil
}

func resolveEngine(model string) string {
	lower := strings.ToLower(model)
	switch {
	case strings.Contains(lower, "sd3") || strings.Contains(lower, "sd-3"):
		return "sd3"
	case strings.Contains(lower, "core"):
		return "core"
	case strings.Contains(lower, "ultra"):
		return "ultra"
	case strings.Contains(lower, "stable-image"):
		return "sd3"
	default:
		return model
	}
}

func sizeToAspectRatio(size string) string {
	switch size {
	case "1024x1024":
		return "1:1"
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
	case "1792x1024":
		return "16:9"
	case "1024x1792":
		return "9:16"
	default:
		return size
	}
}
