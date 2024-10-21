package utils

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type RealAudioExtractor struct{}

func (RealAudioExtractor) ExtractAudio(ctx context.Context, videoFile, audioFile string) (bool, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", videoFile, "-acodec", "pcm_s16le", "-ar", "16000", "-ac", "1", audioFile)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err != nil {
		stderrStr := stderr.String()
		if strings.Contains(stderrStr, "Output file does not contain any stream") {
			return false, nil // No audio stream, but not an error
		}
		return false, fmt.Errorf("ffmpeg error: %v\nStderr: %s", err, stderrStr)
	}

	return true, nil
}
