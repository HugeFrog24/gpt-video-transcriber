package main

import (
	"context"
	"fmt"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

// GenerateDescription sends the transcription to OpenAI GPT-4 and generates a clear and concise description
func GenerateDescription(transcription string) (string, error) {
	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	ctx := context.Background()

	req := openai.ChatCompletionRequest{
		Model: openai.GPT4,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a helpful assistant that generates clear and concise descriptions for videos. Ensure the description is in the same language as the transcription. Write the description from the perspective of the vlogger (HugeFrog24) and correct any misrecognitions of 'HugeFrog24'.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: fmt.Sprintf("Please generate a clear and concise description for the following video transcription:\n\n%s", transcription),
			},
		},
	}

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("error generating description: %v", err)
	}

	return resp.Choices[0].Message.Content, nil
}
