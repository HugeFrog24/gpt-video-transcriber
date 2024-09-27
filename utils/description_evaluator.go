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

type RealDescriptionEvaluator struct{}

func (RealDescriptionEvaluator) EvaluateDescriptions(descriptions []string, transcription string, filename string) (int, error) {
	return EvaluateDescriptions(descriptions, transcription, filename)
}

func EvaluateDescriptions(descriptions []string, transcription string, filename string) (int, error) {
	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	ctx := context.Background()

	// Detect the language of the transcription
	detector := lingua.NewLanguageDetectorBuilder().FromAllLanguages().Build()
	language, _ := detector.DetectLanguageOf(transcription)

	// Adjust the prompt to include the detected language
	prompt := fmt.Sprintf(`You are an expert in evaluating video descriptions in %s. Please analyze the following descriptions and return the number (1-based index) of the best description based on:

- How well it matches the transcription and filename.
- Style, language consistency, and clarity.
- Prioritize descriptions that are in the same language as the transcription.
- Only return the number, no other text.

`, language.String())

	prompt += fmt.Sprintf("Filename: %s\n\n", filename)
	prompt += fmt.Sprintf("Transcription:\n%s\n\n", transcription)
	prompt += "Descriptions:\n"
	for i, desc := range descriptions {
		prompt += fmt.Sprintf("%d. %s\n\n", i+1, desc)
	}

	req := openai.ChatCompletionRequest{
		Model: openai.GPT4,
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
	}

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("error evaluating descriptions: %v", err)
	}

	bestIndex, err := strconv.Atoi(strings.TrimSpace(resp.Choices[0].Message.Content))
	if err != nil {
		return 0, fmt.Errorf("error parsing best description index: %v", err)
	}

	return bestIndex, nil
}
