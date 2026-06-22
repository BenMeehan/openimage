package provider

import (
	"fmt"
)

type GenerateParams struct {
	Prompt  string
	Model   string
	N       int
	Size    string
	Quality string
	Style   string
}

type GenerateResult struct {
	Data          []byte
	RevisedPrompt string
}

type Provider interface {
	GenerateImage(params *GenerateParams) (*GenerateResult, error)
	Name() string
}

func New(name, apiKey, baseURL string) (Provider, error) {
	if name == "" && baseURL == "" {
		name = "openrouter"
	}

	switch name {
	case "openai", "openrouter", "openai-compatible":
		return NewOpenAIProvider(apiKey, baseURL), nil
	case "stability":
		return NewStabilityProvider(apiKey, baseURL), nil
	case "replicate":
		return NewReplicateProvider(apiKey, baseURL), nil
	case "ideogram":
		return NewIdeogramProvider(apiKey, baseURL), nil
	case "deepai":
		return NewDeepAIProvider(apiKey, baseURL), nil
	case "getimg":
		return NewGetImgProvider(apiKey, baseURL), nil
	case "clipdrop":
		return NewClipdropProvider(apiKey, baseURL), nil
	case "segmind":
		return NewSegmindProvider(apiKey, baseURL), nil
	default:
		if baseURL != "" {
			return NewOpenAIProvider(apiKey, baseURL), nil
		}
		return nil, fmt.Errorf(
			"unknown provider: %s (supported: openai, openrouter, stability, replicate, ideogram, deepai, getimg, clipdrop, segmind)",
			name,
		)
	}
}
