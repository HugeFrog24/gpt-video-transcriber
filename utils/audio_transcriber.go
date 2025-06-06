package utils

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	lingua "github.com/pemistahl/lingua-go"
	openai "github.com/sashabaranov/go-openai"
)

type RealAudioTranscriber struct{}

func (RealAudioTranscriber) TranscribeAudio(ctx context.Context, audioFile string, maxDuration time.Duration) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY environment variable is not set")
	}
	client := openai.NewClient(apiKey)

	// Split audio into chunks
	chunks, err := splitAudio(ctx, audioFile, maxDuration) // Pass ctx here
	if err != nil {
		return "", fmt.Errorf("failed to split audio: %v", err)
	}

	// Ensure all temporary chunk files are cleaned up
	for _, chunk := range chunks {
		defer func(chunkPath string) {
			if err := os.Remove(chunkPath); err != nil {
				fmt.Printf("Failed to remove temporary audio chunk %s: %v\n", chunkPath, err)
			}
		}(chunk)
	}

	var fullTranscription strings.Builder
	for _, chunk := range chunks {
		// Transcribe the chunk
		req := openai.AudioRequest{
			Model:    openai.Whisper1,
			FilePath: chunk,
		}
		resp, err := client.CreateTranscription(ctx, req)
		if err != nil {
			return "", fmt.Errorf("transcription error: %v", err)
		}
		fullTranscription.WriteString(resp.Text)
		fullTranscription.WriteString(" ")

		// Temporary file cleanup is handled by defer
	}

	transcription := strings.TrimSpace(fullTranscription.String())

	// Detect the language of the transcription
	detector := lingua.NewLanguageDetectorBuilder().FromAllLanguages().Build()
	language, _ := detector.DetectLanguageOf(transcription)

	// Optionally, log or handle the detected language
	fmt.Printf("Detected transcription language: %s\n", language.String())

	return transcription, nil
}

func splitAudio(ctx context.Context, audioFile string, maxDuration time.Duration) ([]string, error) {
	var chunks []string

	// Get audio duration
	duration, err := getAudioDuration(audioFile)
	if err != nil {
		return nil, err
	}

	// Calculate number of chunks
	numChunks := int(duration.Seconds()/maxDuration.Seconds()) + 1

	for i := 0; i < numChunks; i++ {
		start := time.Duration(i) * maxDuration
		chunkFile := fmt.Sprintf("%s_chunk_%d.wav", strings.TrimSuffix(audioFile, filepath.Ext(audioFile)), i)

		// Sanitize inputs
		audioFileSafe := filepath.Clean(audioFile)
		chunkFileSafe := filepath.Clean(chunkFile)

		// #nosec G204
		cmd := exec.CommandContext(ctx, "ffmpeg", "-i", audioFileSafe, "-ss", fmt.Sprintf("%f", start.Seconds()), "-t", fmt.Sprintf("%f", maxDuration.Seconds()), "-acodec", "pcm_s16le", "-ar", "16000", "-ac", "1", chunkFileSafe)
		err := cmd.Run()
		if err != nil {
			return chunks, fmt.Errorf("failed to create audio chunk: %v", err)
		}

		chunks = append(chunks, chunkFileSafe)
	}

	return chunks, nil
}

func getAudioDuration(audioFile string) (time.Duration, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", audioFile)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get audio duration: %v", err)
	}

	durationStr := strings.TrimSpace(string(output))
	duration, err := time.ParseDuration(fmt.Sprintf("%ss", durationStr))
	if err != nil {
		return 0, fmt.Errorf("failed to parse audio duration: %v", err)
	}

	return duration, nil
}
