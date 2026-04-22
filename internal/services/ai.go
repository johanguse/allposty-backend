package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

type AIService struct {
	client *openai.Client
}

func NewAIService(apiKey string) *AIService {
	return &AIService{client: openai.NewClient(apiKey)}
}

type CaptionInput struct {
	Topic    string   // e.g. "launching a new product"
	Platform string   // instagram | linkedin | twitter | tiktok | facebook | youtube
	Tone     string   // casual | professional | funny | inspirational
	Keywords []string // optional keyword/hashtag hints
	Language string   // "en", "pt", "es" — defaults to "en"
}

type CaptionResult struct {
	Caption  string `json:"caption"`
	Hashtags string `json:"hashtags,omitempty"`
}

func (s *AIService) GenerateCaption(ctx context.Context, input CaptionInput) (*CaptionResult, error) {
	if input.Language == "" {
		input.Language = "en"
	}
	if input.Tone == "" {
		input.Tone = "casual"
	}

	systemPrompt := `You are a social media copywriter.
Write captions that are engaging, platform-appropriate, and optimized for reach.
Always return valid JSON with exactly two fields: "caption" (string) and "hashtags" (string, space-separated hashtags).
Keep captions concise unless the platform expects longer content (LinkedIn, YouTube).`

	resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4o,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: buildCaptionPrompt(input)},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
		Temperature: 0.8,
		MaxTokens:   600,
	})
	if err != nil {
		return nil, fmt.Errorf("openai: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("openai: no response generated")
	}

	var result CaptionResult
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result); err != nil {
		// Fallback: return raw content as-is
		return &CaptionResult{Caption: strings.TrimSpace(resp.Choices[0].Message.Content)}, nil
	}
	return &result, nil
}

var platformHints = map[string]string{
	"instagram": "125-150 chars ideal (max 2200). Use line breaks. 5-10 hashtags at the end.",
	"linkedin":  "Professional tone, up to 1300 chars. Storytelling works well. 3-5 hashtags.",
	"twitter":   "Max 280 chars total including hashtags. Punchy and direct.",
	"tiktok":    "Short and energetic, 150 chars max. 3-5 hashtags.",
	"facebook":  "Conversational, 40-80 chars gets most engagement. End with a question.",
	"youtube":   "Video description: hook in first 2 lines, then details, then hashtags at end.",
}

func buildCaptionPrompt(input CaptionInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Platform: %s\n", input.Platform)
	fmt.Fprintf(&b, "Topic: %s\n", input.Topic)
	fmt.Fprintf(&b, "Tone: %s\n", input.Tone)
	fmt.Fprintf(&b, "Language: %s (write the caption in this language)\n", input.Language)
	if len(input.Keywords) > 0 {
		fmt.Fprintf(&b, "Keywords/themes to incorporate: %s\n", strings.Join(input.Keywords, ", "))
	}
	if hint, ok := platformHints[input.Platform]; ok {
		fmt.Fprintf(&b, "Platform guidelines: %s\n", hint)
	}
	b.WriteString("\nWrite the caption now. Return JSON only.")
	return b.String()
}
