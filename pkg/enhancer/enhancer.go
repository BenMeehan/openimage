package enhancer

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const openRouterChatURL = "https://openrouter.ai/api/v1/chat/completions"

type Enhancer struct {
	APIKey string
	HTTP   *http.Client
}

func New(apiKey string) *Enhancer {
	return &Enhancer{
		APIKey: apiKey,
		HTTP: &http.Client{Timeout: 120 * time.Second},
	}
}

type chatMessage struct {
	Role    string       `json:"role"`
	Content []chatPart   `json:"content"`
}

type chatPart struct {
	Type     string     `json:"type"`
	Text     string     `json:"text,omitempty"`
	ImageURL *imagePart `json:"image_url,omitempty"`
}

type imagePart struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (e *Enhancer) EnhancePrompt(prompt string) (string, error) {
	messages := []chatMessage{
		{
			Role: "system",
			Content: []chatPart{{
				Type: "text",
				Text: `You are an expert AI image prompt engineer. Your job is to enhance the user's prompt for better image generation results. Follow these rules strictly:

1. Add specific details about composition, lighting, color palette, mood, texture, and camera angle.
2. Include style keywords (e.g. cinematic, photorealistic, oil painting, 8k, trending on artstation).
3. Correct grammar and spelling but preserve the user's intent.
4. Keep the enhanced prompt under 200 words.
5. Return ONLY the enhanced prompt text. No explanations, no prefixes, no quotes.`,
			}},
		},
		{
			Role: "user",
			Content: []chatPart{{
				Type: "text",
				Text: fmt.Sprintf("Enhance this prompt for AI image generation:\n\n%s", prompt),
			}},
		},
	}

	return e.callLLM(messages)
}

func (e *Enhancer) RefinePrompt(originalPrompt, userFeedback string) (string, error) {
	messages := []chatMessage{
		{
			Role: "system",
			Content: []chatPart{{
				Type: "text",
				Text: `You are an expert AI image prompt engineer. Given an original prompt and user feedback, create an improved version that incorporates the feedback while preserving the original intent.

Rules:
1. Apply the user's feedback precisely.
2. Maintain or improve composition, lighting, color, mood, and detail.
3. Return ONLY the improved prompt text. No explanations.`,
			}},
		},
		{
			Role: "user",
			Content: []chatPart{{
				Type: "text",
				Text: fmt.Sprintf("Original prompt:\n%s\n\nFeedback: %s\n\nWrite the improved prompt:", originalPrompt, userFeedback),
			}},
		},
	}

	return e.callLLM(messages)
}

func (e *Enhancer) DescribeImage(imageData []byte) (string, error) {
	b64 := base64.StdEncoding.EncodeToString(imageData)
	dataURI := "data:image/png;base64," + b64

	messages := []chatMessage{
		{
			Role: "user",
			Content: []chatPart{
				{Type: "text", Text: "Describe this image in 2–3 sentences. Note the subject, composition, color palette, lighting, mood, and any notable details or issues. Be concise."},
				{Type: "image_url", ImageURL: &imagePart{URL: dataURI}},
			},
		},
	}

	req := chatRequest{
		Model:    "openai/gpt-4o",
		Messages: messages,
	}

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", openRouterChatURL, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+e.APIKey)
	httpReq.Header.Set("User-Agent", "openimage/1.0")

	resp, err := e.HTTP.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("calling vision model: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("vision API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result chatResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parsing vision response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from vision model")
	}

	return result.Choices[0].Message.Content, nil
}

func (e *Enhancer) callLLM(messages []chatMessage) (string, error) {
	req := chatRequest{
		Model:    "openai/gpt-4o",
		Messages: messages,
	}

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", openRouterChatURL, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+e.APIKey)
	httpReq.Header.Set("User-Agent", "openimage/1.0")

	resp, err := e.HTTP.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("calling LLM: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result chatResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parsing LLM response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	return result.Choices[0].Message.Content, nil
}
