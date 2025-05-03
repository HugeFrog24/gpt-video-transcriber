package utils

import (
	"context"
	"fmt"
	"os"

	lingua "github.com/pemistahl/lingua-go"
	openai "github.com/sashabaranov/go-openai"
)

type RealDescriptionGenerator struct{}

func (RealDescriptionGenerator) GenerateDescriptions(transcription string, filename string, attempts int) ([]string, error) {
	return GenerateDescriptions(transcription, filename, attempts)
}

// GenerateDescriptions sends the transcription and filename to OpenAI GPT-4 to generate descriptions
func GenerateDescriptions(transcription string, filename string, attempts int) ([]string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	client := openai.NewClient(apiKey)
	ctx := context.Background()

	// Create a TextSummarizer instance
	summarizer := NewTextSummarizer(client)

	// Summarize the transcription if it's too long
	summarizedTranscription, err := summarizer.SummarizeText(transcription, 2000) // Target 2000 characters
	if err != nil {
		return nil, fmt.Errorf("error summarizing transcription: %v", err)
	}

	// Detect the language of the summarized transcription
	detector := lingua.NewLanguageDetectorBuilder().FromAllLanguages().Build()
	language, reliable := detector.DetectLanguageOf(summarizedTranscription)
	if !reliable {
		fmt.Printf("Warning: Language detection may not be reliable for this text\n")
	}

	// Adjust the prompt to include the detected language
	systemPrompt := fmt.Sprintf("You are a helpful assistant that generates clear and concise descriptions for videos in %s. Ensure the description is in the same language as the transcription. Write the description from the perspective of the vlogger (HugeFrog24) and correct any misrecognitions of 'HugeFrog24'. Use the filename to infer additional context about the video's content or theme, as it may contain relevant keywords or information not present in the transcription.", language.String())

	const maxDescriptionLength = 1000

	descriptions := make([]string, 0, attempts)

	for i := 0; i < attempts; i++ {
		req := openai.ChatCompletionRequest{
			Model: openai.GPT4,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: fmt.Sprintf("Based on the following transcription and filename, generate a clear and concise description for the video (maximum %d characters).\n\nFilename: %s\n\nTranscription:\n%s", maxDescriptionLength, filename, summarizedTranscription),
				},
			},
			MaxTokens: maxDescriptionLength,
		}

		resp, err := client.CreateChatCompletion(ctx, req)
		if err != nil {
			return descriptions, fmt.Errorf("error generating description: %v", err)
		}

		description := resp.Choices[0].Message.Content
		if len(description) > maxDescriptionLength {
			description = description[:maxDescriptionLength]
		}

		descriptions = append(descriptions, description)
	}

	return descriptions, nil
}
