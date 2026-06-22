package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultReplicateBaseURL = "https://api.replicate.com/v1"

type ReplicateProvider struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

func NewReplicateProvider(apiKey, baseURL string) *ReplicateProvider {
	if baseURL == "" {
		baseURL = defaultReplicateBaseURL
	}
	baseURL = trimSlash(baseURL)
	return &ReplicateProvider{
		APIKey:  apiKey,
		BaseURL: baseURL,
		HTTP: &http.Client{
			Timeout: 180 * time.Second,
		},
	}
}

func (p *ReplicateProvider) Name() string { return "replicate" }

type replicateRequest struct {
	Version string         `json:"version,omitempty"`
	Input   map[string]any `json:"input"`
}

type replicateResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Output any    `json:"output"`
	Error  string `json:"error"`
}

func (p *ReplicateProvider) GenerateImage(params *GenerateParams) (*GenerateResult, error) {
	owner, model, version := parseReplicateModel(params.Model)

	endpoint := fmt.Sprintf("%s/models/%s/%s/predictions", p.BaseURL, owner, model)

	input := map[string]any{
		"prompt": params.Prompt,
	}

	if params.Size != "" {
		w, h := parseSize(params.Size)
		if w > 0 && h > 0 {
			input["width"] = w
			input["height"] = h
		}
	}
	if params.N > 1 {
		input["num_outputs"] = params.N
	}

	reqBody := replicateRequest{
		Version: version,
		Input:   input,
	}

	body, _ := json.Marshal(reqBody)

	httpReq, err := http.NewRequest("POST", endpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	httpReq.Header.Set("Prefer", "wait")
	httpReq.Header.Set("User-Agent", "openimage/1.0")

	resp, err := p.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("Replicate API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result replicateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("Replicate error: %s", result.Error)
	}

	if result.Status != "succeeded" && result.Status != "completed" {
		return nil, fmt.Errorf("Replicate generation not complete (status: %s)", result.Status)
	}

	imageURL := extractURL(result.Output)
	if imageURL == "" {
		return nil, fmt.Errorf("no image URL in response")
	}

	imageData, err := downloadImage(p.HTTP, imageURL)
	if err != nil {
		return nil, fmt.Errorf("downloading image: %w", err)
	}

	return &GenerateResult{Data: imageData}, nil
}

func parseReplicateModel(model string) (owner, name, version string) {
	parts := strings.SplitN(model, ":", 2)
	id := parts[0]
	if len(parts) == 2 {
		version = parts[1]
	}
	ownerParts := strings.SplitN(id, "/", 2)
	if len(ownerParts) == 2 {
		owner = ownerParts[0]
		name = ownerParts[1]
	} else {
		owner = ownerParts[0]
	}
	return
}

func extractURL(output any) string {
	switch v := output.(type) {
	case string:
		return v
	case []any:
		if len(v) > 0 {
			if s, ok := v[0].(string); ok {
				return s
			}
		}
	}
	return ""
}



