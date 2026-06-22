package provider

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

const defaultZhipuBaseURL = "https://open.bigmodel.cn/api/paas/v4"

type ZhipuProvider struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

func NewZhipuProvider(apiKey, baseURL string) *ZhipuProvider {
	if baseURL == "" {
		baseURL = defaultZhipuBaseURL
	}
	baseURL = trimSlash(baseURL)
	return &ZhipuProvider{
		APIKey:  apiKey,
		BaseURL: baseURL,
		HTTP: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (p *ZhipuProvider) Name() string { return "zhipu" }

type zhipuRequest struct {
	Model    string          `json:"model"`
	Messages []zhipuMessage  `json:"messages"`
}

type zhipuMessage struct {
	Role    string          `json:"role"`
	Content []zhipuContent  `json:"content"`
}

type zhipuContent struct {
	Type     string         `json:"type"`
	Text     string         `json:"text,omitempty"`
	ImageURL *zhipuImageURL `json:"image_url,omitempty"`
}

type zhipuImageURL struct {
	URL string `json:"url"`
}

type zhipuResponse struct {
	Choices []zhipuChoice `json:"choices"`
}

type zhipuChoice struct {
	Message zhipuRespMessage `json:"message"`
}

type zhipuRespMessage struct {
	Role    string           `json:"role"`
	Content []zhipuRespPart  `json:"content"`
}

type zhipuRespPart struct {
	Type     string           `json:"type"`
	ImageURL *zhipuRespImage  `json:"image_url,omitempty"`
}

type zhipuRespImage struct {
	URL string `json:"url"`
}

var dataURIPattern = regexp.MustCompile(`^data:image/\w+;base64,`)

func (p *ZhipuProvider) GenerateImage(params *GenerateParams) (*GenerateResult, error) {
	model := params.Model
	if model == "" {
		model = "cogview-3-plus"
	}

	reqBody := zhipuRequest{
		Model: model,
		Messages: []zhipuMessage{{
			Role: "user",
			Content: []zhipuContent{{
				Type: "text",
				Text: params.Prompt,
			}},
		}},
	}

	body, _ := json.Marshal(reqBody)

	httpReq, err := http.NewRequest("POST", p.BaseURL+"/chat/completions", bytes.NewReader(body))
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
		return nil, fmt.Errorf("Zhipu API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result zhipuResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no response from model")
	}

	for _, part := range result.Choices[0].Message.Content {
		if part.Type == "image_url" && part.ImageURL != nil {
			raw := part.ImageURL.URL

			if dataURIPattern.MatchString(raw) {
				b64 := dataURIPattern.ReplaceAllString(raw, "")
				data, err := base64.StdEncoding.DecodeString(b64)
				if err != nil {
					return nil, fmt.Errorf("decoding base64 image: %w", err)
				}
				return &GenerateResult{Data: data}, nil
			}

			data, err := downloadImage(p.HTTP, raw)
			if err != nil {
				return nil, fmt.Errorf("downloading image: %w", err)
			}
			return &GenerateResult{Data: data}, nil
		}
	}

	return nil, fmt.Errorf("no image in response")
}
