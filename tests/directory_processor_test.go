package tests

import (
	"context"
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HugeFrog24/gpt-video-transcriber/utils"
)

func TestProcessDirectory(t *testing.T) {
	ctx := context.Background()
	testDir, err := os.MkdirTemp("", "test-directory")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(testDir); err != nil {
			t.Logf("Failed to remove test directory: %v", err)
		}
	}()

	outputXML := filepath.Join(testDir, "test_output.xml")

	// Create multiple test video files
	videoFiles := []string{"test_video1.mp4", "test_video2.mp4", "non_video_file.txt"}
	for _, file := range videoFiles {
		filePath := filepath.Join(testDir, file)
		if err := os.WriteFile(filePath, []byte("mock content"), 0644); err != nil {
			t.Fatalf("Failed to create mock file %s: %v", file, err)
		}
	}

	// Create .tmp directory
	if err := os.MkdirAll(".tmp", os.ModePerm); err != nil {
		t.Fatalf("Failed to create .tmp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(".tmp"); err != nil {
			t.Logf("Failed to remove .tmp directory: %v", err)
		}
	}() // Clean up after test

	// Mock implementations
	mockExtractor := &utils.MockAudioExtractor{
		ExtractAudioFunc: func(ctx context.Context, videoFile, audioFile string) (bool, error) {
			// Create a mock audio file
			return true, os.WriteFile(audioFile, []byte("mock audio content"), 0644)
		},
	}
	mockTranscriber := &utils.MockAudioTranscriber{
		TranscribeAudioFunc: func(ctx context.Context, audioFile string, maxDuration time.Duration) (string, error) {
			return "Mock transcription", nil
		},
	}
	mockGenerator := &utils.MockDescriptionGenerator{
		GenerateDescriptionsFunc: func(transcription string, filename string, attempts int) ([]string, error) {
			return []string{"Mock description 1", "Mock description 2"}, nil
		},
	}
	mockEvaluator := &utils.MockDescriptionEvaluator{
		EvaluateDescriptionsFunc: func(descriptions []string, transcription string, filename string) (int, error) {
			return 1, nil
		},
	}

	results, err := utils.ProcessDirectory(
		ctx,
		testDir,
		outputXML,
		2,
		mockExtractor,
		mockTranscriber,
		mockGenerator,
		mockEvaluator,
	)
	if err != nil {
		t.Fatalf("ProcessDirectory failed: %v", err)
	}

	// Check the number of processed files
	expectedProcessedFiles := 2 // Only .mp4 files should be processed
	if len(results.Results) != expectedProcessedFiles {
		t.Errorf("Expected %d processed files, got %d", expectedProcessedFiles, len(results.Results))
	}

	// Verify the content of the results
	for _, result := range results.Results {
		if result.Transcription != "Mock transcription" {
			t.Errorf("Expected transcription 'Mock transcription', got '%s'", result.Transcription)
		}
		if len(result.Descriptions) != 2 {
			t.Errorf("Expected 2 descriptions, got %d", len(result.Descriptions))
		}
		if result.BestDescriptionIndex != 1 {
			t.Errorf("Expected best description index 1, got %d", result.BestDescriptionIndex)
		}
	}

	// Verify XML output
	xmlContent, err := os.ReadFile(outputXML)
	if err != nil {
		t.Fatalf("Failed to read output XML: %v", err)
	}

	var parsedResults utils.TranscriptionResults
	if err := xml.Unmarshal(xmlContent, &parsedResults); err != nil {
		t.Fatalf("Failed to parse output XML: %v", err)
	}

	if len(parsedResults.Results) != expectedProcessedFiles {
		t.Errorf("XML: Expected %d processed files, got %d", expectedProcessedFiles, len(parsedResults.Results))
	}

	// Remove the output XML to simulate a fresh run
	if err := os.Remove(outputXML); err != nil && !os.IsNotExist(err) {
		t.Fatalf("Failed to remove output XML: %v", err)
	}

	// Second test run with error-inducing extractor
	errorExtractor := &utils.MockAudioExtractor{
		ExtractAudioFunc: func(ctx context.Context, videoFile, audioFile string) (bool, error) {
			return false, os.ErrNotExist // Simulate an error
		},
	}

	_, err = utils.ProcessDirectory(
		ctx,
		testDir,
		outputXML,
		2,
		errorExtractor,
		mockTranscriber,
		mockGenerator,
		mockEvaluator,
	)

	if err == nil {
		t.Error("Expected an error when audio extraction fails, got nil")
	}
}
