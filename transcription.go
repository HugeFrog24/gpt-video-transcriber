package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

func extractAudio(ctx context.Context, videoFile, audioFile string) error {
	if strings.Contains(videoFile, " ") || strings.Contains(audioFile, " ") {
		return fmt.Errorf("invalid file path")
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", videoFile, "-acodec", "pcm_s16le", "-ar", "16000", "-ac", "1", audioFile)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func transcribeAudio(audioFile string) (string, error) {
	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	ctx := context.Background()

	req := openai.AudioRequest{
		Model:    openai.Whisper1,
		FilePath: audioFile,
	}
	resp, err := client.CreateTranscription(ctx, req)
	if err != nil {
		return "", fmt.Errorf("transcription error: %v", err)
	}
	return resp.Text, nil
}
