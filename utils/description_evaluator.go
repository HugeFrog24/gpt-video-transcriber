package utils

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pemistahl/lingua-go"
	openai "github.com/sashabaranov/go-openai"
)

type RealDescriptionEvaluator struct {
	client *openai.Client
}

func NewRealDescriptionEvaluator() (*RealDescriptionEvaluator, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is not set")
	}

	client := openai.NewClient(apiKey)
	return &RealDescriptionEvaluator{
		client: client,
	}, nil
}

func (e *RealDescriptionEvaluator) EvaluateDescriptions(descriptions []string, transcription string, filename string) (int, error) {
	ctx := context.Background()

	// Detect the language of the transcription
	detector := lingua.NewLanguageDetectorBuilder().FromAllLanguages().Build()
	language, _ := detector.DetectLanguageOf(transcription)

	prompt := fmt.Sprintf(`You are an expert in evaluating video descriptions in %s. Analyze the following descriptions and return the number (1-based index) of the best description based on:

- How well it matches the transcription and filename.
- Style, language consistency, and clarity.
- Prioritize descriptions that are in the same language as the transcription.
- Only return the number, no other text.

Filename: %s

Transcription:
%s

Descriptions:
%s

Remember, respond with ONLY the number of the best description, nothing else.`, language.String(), filename, transcription, formatDescriptions(descriptions))

	for attempts := 0; attempts < 3; attempts++ {
		req := openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo16K,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a helpful assistant that evaluates video descriptions.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			MaxTokens: 10,
		}

		resp, err := e.client.CreateChatCompletion(ctx, req)
		if err != nil {
			return 0, fmt.Errorf("error evaluating descriptions: %v", err)
		}

		content := strings.TrimSpace(resp.Choices[0].Message.Content)
		bestIndex, err := strconv.Atoi(content)
		if err == nil && bestIndex > 0 && bestIndex <= len(descriptions) {
			return bestIndex, nil
		}

		// If we didn't get a valid number, append clarification without overwriting
		prompt += "\nRemember, respond with ONLY the number of the best description, nothing else."
	}

	return 0, fmt.Errorf("failed to get a valid response after multiple attempts")
}

func formatDescriptions(descriptions []string) string {
	var result strings.Builder
	for i, desc := range descriptions {
		result.WriteString(fmt.Sprintf("%d. %s\n\n", i+1, desc))
	}
	return result.String()
}
