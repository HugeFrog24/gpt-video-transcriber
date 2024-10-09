package utils

import (
	"context"
	"time"
)

type AudioExtractor interface {
	ExtractAudio(ctx context.Context, videoFile, audioFile string) (bool, error)
}

type AudioTranscriber interface {
	TranscribeAudio(ctx context.Context, audioFile string, maxDuration time.Duration) (string, error)
}

type DescriptionGenerator interface {
	GenerateDescriptions(transcription string, filename string, attempts int) ([]string, error)
}

type DescriptionEvaluator interface {
	EvaluateDescriptions(descriptions []string, transcription string, filename string) (int, error)
}
