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

const defaultSegmindBaseURL = "https://api.segmind.com"

type SegmindProvider struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

func NewSegmindProvider(apiKey, baseURL string) *SegmindProvider {
	if baseURL == "" {
		baseURL = defaultSegmindBaseURL
	}
	baseURL = trimSlash(baseURL)
	return &SegmindProvider{
		APIKey:  apiKey,
		BaseURL: baseURL,
		HTTP: &http.Client{
			Timeout: 180 * time.Second,
		},
	}
}

func (p *SegmindProvider) Name() string { return "segmind" }

type segmindRequest struct {
	Prompt   string `json:"prompt"`
	Samples  int    `json:"samples,omitempty"`
	Width    int    `json:"img_width,omitempty"`
	Height   int    `json:"img_height,omitempty"`
	Seed     int64  `json:"seed,omitempty"`
	Base64   bool   `json:"base64"`
}

type segmindV2SubmitResponse struct {
	RequestID  string `json:"request_id"`
	StatusURL  string `json:"status_url"`
	ResponseURL string `json:"response_url"`
}

type segmindV2ResultResponse struct {
	Status string `json:"status"`
	Output string `json:"output"`
}

func (p *SegmindProvider) GenerateImage(params *GenerateParams) (*GenerateResult, error) {
	model := params.Model
	if model == "" {
		model = "sdxl1.0-newreality-lightning"
	}

	w, h := 1024, 1024
	if params.Size != "" {
		w, h = parseSize(params.Size)
		if w == 0 {
			w = 1024
		}
		if h == 0 {
			h = 1024
		}
	}

	reqBody := segmindRequest{
		Prompt:  params.Prompt,
		Samples: params.N,
		Width:   w,
		Height:  h,
		Seed:    -1,
		Base64:  true,
	}

	body, _ := json.Marshal(reqBody)

	httpReq, err := http.NewRequest("POST", p.BaseURL+"/v1/"+model, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.APIKey)

	resp, err := p.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusAccepted {
		return p.handleAsyncV2(model, body)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Segmind API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	respBody, _ := io.ReadAll(resp.Body)

	if len(respBody) > 0 && respBody[0] == '{' {
		imageData, err := extractSegmindImage(respBody)
		if err == nil && len(imageData) > 0 {
			return &GenerateResult{Data: imageData}, nil
		}
	}

	return &GenerateResult{Data: respBody}, nil
}

func (p *SegmindProvider) handleAsyncV2(model string, reqBody []byte) (*GenerateResult, error) {
	submitReq, _ := http.NewRequest("POST", p.BaseURL+"/v2/"+model, bytes.NewReader(reqBody))
	submitReq.Header.Set("Content-Type", "application/json")
	submitReq.Header.Set("x-api-key", p.APIKey)

	resp, err := p.HTTP.Do(submitReq)
	if err != nil {
		return nil, fmt.Errorf("submitting async request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var submit segmindV2SubmitResponse
	if err := json.Unmarshal(respBody, &submit); err != nil {
		return nil, fmt.Errorf("parsing submit response: %w", err)
	}

	for i := 0; i < 60; i++ {
		time.Sleep(2 * time.Second)

		statusReq, _ := http.NewRequest("GET", submit.ResponseURL, nil)
		statusReq.Header.Set("x-api-key", p.APIKey)

		statusResp, err := p.HTTP.Do(statusReq)
		if err != nil {
			continue
		}
		statusBody, _ := io.ReadAll(statusResp.Body)
		statusResp.Body.Close()

		var result segmindV2ResultResponse
		if err := json.Unmarshal(statusBody, &result); err != nil {
			continue
		}

		if result.Status == "COMPLETED" && result.Output != "" {
			imageData, err := downloadImage(p.HTTP, result.Output)
			if err != nil {
				return nil, fmt.Errorf("downloading image: %w", err)
			}
			return &GenerateResult{Data: imageData}, nil
		}

		if result.Status == "FAILED" {
			return nil, fmt.Errorf("Segmind generation failed")
		}
	}

	return nil, fmt.Errorf("Segmind generation timed out")
}

func extractSegmindImage(data []byte) ([]byte, error) {
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	for _, key := range []string{"image", "output", "data", "images"} {
		if v, ok := result[key]; ok {
			switch val := v.(type) {
			case string:
				return base64.StdEncoding.DecodeString(val)
			case []any:
				if len(val) > 0 {
					if s, ok := val[0].(string); ok {
						return base64.StdEncoding.DecodeString(s)
					}
					if m, ok := val[0].(map[string]any); ok {
						if url, ok := m["url"].(string); ok {
							return downloadImage(&http.Client{Timeout: 60 * time.Second}, url)
						}
					}
				}
			}
		}
	}
	return nil, fmt.Errorf("no image in response")
}
