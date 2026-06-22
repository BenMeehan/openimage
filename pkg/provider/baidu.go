package provider

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultBaiduBaseURL = "https://aip.baidubce.com"

type BaiduProvider struct {
	APIKey    string
	SecretKey string
	BaseURL   string
	HTTP      *http.Client
}

func NewBaiduProvider(apiKey, baseURL string) *BaiduProvider {
	if baseURL == "" {
		baseURL = defaultBaiduBaseURL
	}
	baseURL = trimSlash(baseURL)

	clientID := apiKey
	secretKey := ""

	if parts := strings.SplitN(apiKey, ":", 2); len(parts) == 2 {
		clientID = parts[0]
		secretKey = parts[1]
	}

	return &BaiduProvider{
		APIKey:    clientID,
		SecretKey: secretKey,
		BaseURL:   baseURL,
		HTTP: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (p *BaiduProvider) Name() string { return "baidu" }

type baiduTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

type baiduImageRequest struct {
	Prompt       string `json:"prompt"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
	Style        string `json:"style,omitempty"`
	Resolution   string `json:"resolution,omitempty"`
	Num          int    `json:"num"`
	SamplingSteps int   `json:"samplingSteps,omitempty"`
	Seed         int64  `json:"seed,omitempty"`
}

type baiduImageResponse struct {
	Data  []baiduImageData `json:"data"`
	Error int              `json:"error_code"`
	Msg   string           `json:"error_msg"`
}

type baiduImageData struct {
	Object string `json:"object"`
	B64    string `json:"b64_image"`
	Index  int    `json:"index"`
}

func (p *BaiduProvider) getAccessToken() (string, error) {
	if p.SecretKey == "" {
		return p.APIKey, nil
	}

	url := fmt.Sprintf("%s/oauth/2.0/token?grant_type=client_credentials&client_id=%s&client_secret=%s",
		p.BaseURL, p.APIKey, p.SecretKey)

	resp, err := p.HTTP.Get(url)
	if err != nil {
		return "", fmt.Errorf("getting oauth token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var tokenResp baiduTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("parsing token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("no access token in response: %s", string(body))
	}

	return tokenResp.AccessToken, nil
}

func (p *BaiduProvider) GenerateImage(params *GenerateParams) (*GenerateResult, error) {
	token, err := p.getAccessToken()
	if err != nil {
		return nil, err
	}

	model := params.Model
	if model == "" {
		model = "sd_xl"
	}

	endpoint := fmt.Sprintf("%s/rpc/2.0/ai_custom/v1/wenxinworkshop/text2image/%s?access_token=%s",
		p.BaseURL, model, token)

	num := params.N
	if num < 1 {
		num = 1
	}

	res := parseResolution(params.Size)
	reqBody := baiduImageRequest{
		Prompt:     params.Prompt,
		Resolution: res,
		Num:        num,
		Seed:       -1,
	}

	if params.Style != "" {
		reqBody.Style = params.Style
	}

	body, _ := json.Marshal(reqBody)

	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result baiduImageResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if result.Error != 0 {
		return nil, fmt.Errorf("Baidu API error (code %d): %s", result.Error, result.Msg)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no images returned")
	}

	imageData, err := base64.StdEncoding.DecodeString(result.Data[0].B64)
	if err != nil {
		return nil, fmt.Errorf("decoding image: %w", err)
	}

	return &GenerateResult{Data: imageData}, nil
}

func parseResolution(size string) string {
	if size == "" {
		return "1024*1024"
	}
	var w, h int
	fmt.Sscanf(size, "%dx%d", &w, &h)
	if w > 0 && h > 0 {
		return fmt.Sprintf("%d*%d", w, h)
	}
	return "1024*1024"
}
