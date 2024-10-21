package utils

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/pemistahl/lingua-go"
	openai "github.com/sashabaranov/go-openai"
)

const (
	maxChunkSize    = 8000
	targetChunkSize = 4000
	maxIterations   = 10
)

type TextSummarizer struct {
	client *openai.Client
}

func NewTextSummarizer(client *openai.Client) *TextSummarizer {
	return &TextSummarizer{client: client}
}

func (ts *TextSummarizer) SummarizeText(text string, targetLength int) (string, error) {
	return ts.summarizeTextRecursive(text, targetLength, 0)
}

func (ts *TextSummarizer) summarizeTextRecursive(text string, targetLength int, iteration int) (string, error) {
	if len(text) <= targetLength || iteration >= maxIterations {
		return text, nil
	}

	fmt.Printf("Summarization iteration %d: Input length %d characters\n", iteration, len(text))

	chunks := ts.splitTextIntoChunks(text, maxChunkSize)
	summarizedChunks := make([]string, 0, len(chunks))

	for i, chunk := range chunks {
		summary, err := ts.summarizeChunk(chunk)
		if err != nil {
			return "", fmt.Errorf("error summarizing chunk %d: %v", i, err)
		}
		summarizedChunks = append(summarizedChunks, summary)
	}

	combinedSummary := strings.Join(summarizedChunks, " ")
	fmt.Printf("Summarization iteration %d: Output length %d characters\n", iteration, len(combinedSummary))

	if len(combinedSummary) > targetLength {
		return ts.summarizeTextRecursive(combinedSummary, targetLength, iteration+1)
	}

	return combinedSummary, nil
}

func (ts *TextSummarizer) summarizeChunk(chunk string) (string, error) {
	ctx := context.Background()

	// Detect the language of the chunk
	detector := lingua.NewLanguageDetectorBuilder().FromAllLanguages().Build()
	language, _ := detector.DetectLanguageOf(chunk)

	prompt := fmt.Sprintf("Summarize the following text in %s, maintaining key information and context:\n\n%s", language.String(), chunk)

	req := openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: fmt.Sprintf("You are a helpful assistant that summarizes text concisely while retaining key information. Always respond in %s.", language.String()),
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		MaxTokens: 500,
	}

	resp, err := ts.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("error creating chat completion: %v", err)
	}

	return resp.Choices[0].Message.Content, nil
}

func (ts *TextSummarizer) splitTextIntoChunks(text string, chunkSize int) []string {
	words := strings.Fields(text)
	wordsPerChunk := int(math.Ceil(float64(len(words)) / math.Ceil(float64(len(text))/float64(chunkSize))))

	var chunks []string
	for i := 0; i < len(words); i += wordsPerChunk {
		end := i + wordsPerChunk
		if end > len(words) {
			end = len(words)
		}
		chunks = append(chunks, strings.Join(words[i:end], " "))
	}

	return chunks
}
