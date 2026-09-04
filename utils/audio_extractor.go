package utils

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type RealAudioExtractor struct{}

func (RealAudioExtractor) ExtractAudio(ctx context.Context, videoFile, audioFile string) (bool, error) {
	// Clean before checking: filepath.Clean strips a leading "./" and can expose a "-".
	videoFileSafe := filepath.Clean(videoFile)
	audioFileSafe := filepath.Clean(audioFile)
	if err := checkMediaPath("video", videoFileSafe); err != nil {
		return false, err
	}
	if err := checkMediaPath("audio", audioFileSafe); err != nil {
		return false, err
	}

	// #nosec G204 -- the binary is a fixed literal and both paths are checked above;
	// exec.CommandContext passes argv to the OS directly, so no shell interprets them.
	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", videoFileSafe, "-acodec", "pcm_s16le", "-ar", "16000", "-ac", "1", audioFileSafe)
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
