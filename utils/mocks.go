package utils

import (
	"context"
	"time"
)

type MockAudioExtractor struct {
	ExtractAudioFunc func(ctx context.Context, videoFile, audioFile string) (bool, error)
}

func (m *MockAudioExtractor) ExtractAudio(ctx context.Context, videoFile, audioFile string) (bool, error) {
	return m.ExtractAudioFunc(ctx, videoFile, audioFile)
}

type MockAudioTranscriber struct {
	TranscribeAudioFunc func(ctx context.Context, audioFile string, maxDuration time.Duration) (string, error)
}

func (m *MockAudioTranscriber) TranscribeAudio(ctx context.Context, audioFile string, maxDuration time.Duration) (string, error) {
	return m.TranscribeAudioFunc(ctx, audioFile, maxDuration)
}

type MockDescriptionGenerator struct {
	GenerateDescriptionsFunc func(transcription string, filename string, attempts int) ([]string, error)
}

func (m *MockDescriptionGenerator) GenerateDescriptions(transcription string, filename string, attempts int) ([]string, error) {
	return m.GenerateDescriptionsFunc(transcription, filename, attempts)
}

type MockDescriptionEvaluator struct {
	EvaluateDescriptionsFunc func(descriptions []string, transcription string, filename string) (int, error)
}

func (m *MockDescriptionEvaluator) EvaluateDescriptions(descriptions []string, transcription string, filename string) (int, error) {
	return m.EvaluateDescriptionsFunc(descriptions, transcription, filename)
}
