package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultDashScopeBaseURL = "https://dashscope.aliyuncs.com/api/v1"

type DashScopeProvider struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

func NewDashScopeProvider(apiKey, baseURL string) *DashScopeProvider {
	if baseURL == "" {
		baseURL = defaultDashScopeBaseURL
	}
	baseURL = trimSlash(baseURL)
	return &DashScopeProvider{
		APIKey:  apiKey,
		BaseURL: baseURL,
		HTTP: &http.Client{
			Timeout: 180 * time.Second,
		},
	}
}

func (p *DashScopeProvider) Name() string { return "dashscope" }

type dashScopeRequest struct {
	Model      string               `json:"model"`
	Input      dashScopeInput       `json:"input"`
	Parameters dashScopeParameters  `json:"parameters"`
}

type dashScopeInput struct {
	Messages []dashScopeMessage `json:"messages,omitempty"`
	Prompt   string             `json:"prompt,omitempty"`
}

type dashScopeMessage struct {
	Role    string               `json:"role"`
	Content []dashScopeContent   `json:"content"`
}

type dashScopeContent struct {
	Text string `json:"text,omitempty"`
}

type dashScopeParameters struct {
	Size         string `json:"size,omitempty"`
	N            int    `json:"n,omitempty"`
	PromptExtend bool   `json:"prompt_extend"`
	Watermark    bool   `json:"watermark"`
}

type dashScopeCreateResponse struct {
	Output struct {
		TaskStatus string `json:"task_status"`
		TaskID     string `json:"task_id"`
		Choices    []dashScopeChoice `json:"choices"`
	} `json:"output"`
}

type dashScopeChoice struct {
	Message struct {
		Content []dashScopeImageContent `json:"content"`
	} `json:"message"`
}

type dashScopeImageContent struct {
	Image string `json:"image,omitempty"`
	Type  string `json:"type"`
}

type dashScopeQueryResponse struct {
	Output struct {
		TaskStatus string              `json:"task_status"`
		TaskID     string              `json:"task_id"`
		Choices    []dashScopeChoice   `json:"choices"`
		Results    []dashScopeResult   `json:"results"`
	} `json:"output"`
}

type dashScopeResult struct {
	URL string `json:"url"`
}

func (p *DashScopeProvider) GenerateImage(params *GenerateParams) (*GenerateResult, error) {
	model := params.Model
	if model == "" {
		model = "wan2.6-t2i"
	}

	isKling := strings.HasPrefix(model, "kling/")
	isWan26 := strings.Contains(model, "wan2.6")

	if isWan26 {
		return p.generateWan26Sync(params, model)
	}

	if isKling {
		return p.generateAsync(params, model, true)
	}

	return p.generateAsync(params, model, false)
}

func (p *DashScopeProvider) generateWan26Sync(params *GenerateParams, model string) (*GenerateResult, error) {
	reqBody := dashScopeRequest{
		Model: model,
		Input: dashScopeInput{
			Messages: []dashScopeMessage{{
				Role: "user",
				Content: []dashScopeContent{{
					Text: params.Prompt,
				}},
			}},
		},
		Parameters: dashScopeParameters{
			Size:         params.Size,
			N:            params.N,
			PromptExtend: true,
			Watermark:    false,
		},
	}

	body, _ := json.Marshal(reqBody)

	httpReq, err := http.NewRequest("POST", p.BaseURL+"/services/aigc/multimodal-generation/generation", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)

	resp, err := p.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DashScope API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result dashScopeCreateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	for _, choice := range result.Output.Choices {
		for _, content := range choice.Message.Content {
			if content.Image != "" {
				data, err := downloadImage(p.HTTP, content.Image)
				if err != nil {
					return nil, fmt.Errorf("downloading image: %w", err)
				}
				return &GenerateResult{Data: data}, nil
			}
		}
	}

	return nil, fmt.Errorf("no image in response")
}

func (p *DashScopeProvider) generateAsync(params *GenerateParams, model string, isKling bool) (*GenerateResult, error) {
	endpoint := p.BaseURL + "/services/aigc/image-generation/generation"
	if !isKling {
		endpoint = p.BaseURL + "/services/aigc/text2image/image-synthesis"
	}

	var reqBody any
	if isKling {
		reqBody = dashScopeRequest{
			Model: model,
			Input: dashScopeInput{
				Messages: []dashScopeMessage{{
					Role: "user",
					Content: []dashScopeContent{{
						Text: params.Prompt,
					}},
				}},
			},
			Parameters: dashScopeParameters{
				N:         params.N,
				Size:      params.Size,
				PromptExtend: true,
				Watermark:    false,
			},
		}
	} else {
		oldReq := map[string]any{
			"model": model,
			"input": map[string]any{
				"prompt": params.Prompt,
			},
			"parameters": map[string]any{
				"size":     params.Size,
				"n":        params.N,
				"watermark": false,
			},
		}
		body, _ := json.Marshal(oldReq)
		_ = body
		reqBody = oldReq
	}

	body, _ := json.Marshal(reqBody)

	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	httpReq.Header.Set("X-DashScope-Async", "enable")

	resp, err := p.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var createResp dashScopeCreateResponse
	if err := json.Unmarshal(respBody, &createResp); err != nil {
		return nil, fmt.Errorf("parsing create response: %w", err)
	}

	if createResp.Output.TaskID == "" {
		return nil, fmt.Errorf("no task ID in response: %s", string(respBody))
	}

	taskID := createResp.Output.TaskID

	for i := 0; i < 60; i++ {
		time.Sleep(3 * time.Second)

		queryReq, _ := http.NewRequest("GET", p.BaseURL+"/tasks/"+taskID, nil)
		queryReq.Header.Set("Authorization", "Bearer "+p.APIKey)

		queryResp, err := p.HTTP.Do(queryReq)
		if err != nil {
			continue
		}
		queryBody, _ := io.ReadAll(queryResp.Body)
		queryResp.Body.Close()

		var result dashScopeQueryResponse
		if err := json.Unmarshal(queryBody, &result); err != nil {
			continue
		}

		if result.Output.TaskStatus == "FAILED" {
			return nil, fmt.Errorf("DashScope generation failed")
		}

		if result.Output.TaskStatus == "SUCCEEDED" {
			for _, choice := range result.Output.Choices {
				for _, content := range choice.Message.Content {
					if content.Image != "" {
						data, err := downloadImage(p.HTTP, content.Image)
						if err != nil {
							return nil, fmt.Errorf("downloading image: %w", err)
						}
						return &GenerateResult{Data: data}, nil
					}
				}
			}

			for _, r := range result.Output.Results {
				if r.URL != "" {
					data, err := downloadImage(p.HTTP, r.URL)
					if err != nil {
						return nil, fmt.Errorf("downloading image: %w", err)
					}
					return &GenerateResult{Data: data}, nil
				}
			}

			return nil, fmt.Errorf("no image in completed response")
		}
	}

	return nil, fmt.Errorf("DashScope generation timed out (3 minutes)")
}
