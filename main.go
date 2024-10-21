package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/HugeFrog24/gpt-video-transcriber/utils"

	"flag"

	"github.com/joho/godotenv"
)

const defaultDescriptionAttempts = 3

func main() {
	// Define command-line flags
	descriptionCount := flag.Int("descriptions", defaultDescriptionAttempts, "Number of descriptions to generate for each video")
	flag.Parse()

	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Create .tmp directory if it doesn't exist
	tmpDir := ".tmp"
	if err := os.MkdirAll(tmpDir, 0750); err != nil {
		log.Fatalf("Failed to create .tmp directory: %v", err)
	}

	// Clean up .tmp directory at startup
	cleanupTmpDir(tmpDir)

	if flag.NArg() < 1 {
		log.Fatal("Usage: go run main.go [-descriptions <number>] \"<video_file_path_or_directory>\"")
	}
	inputPath := flag.Arg(0)

	// Create a context that is cancelled on interrupt signal
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signal for cleanup
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nReceived interrupt signal, cleaning up...")
		cancel()
		cleanupTmpDir(tmpDir)
		os.Exit(1)
	}()

	// Check if the input path is a directory or a file
	info, err := os.Stat(inputPath)
	if err != nil {
		log.Fatalf("Failed to stat input path: %v", err)
	}

	if info.IsDir() {
		// Process directory
		outputXML := "transcription_results.xml"
		evaluator, err := utils.NewRealDescriptionEvaluator()
		if err != nil {
			log.Fatalf("Failed to create description evaluator: %v", err)
		}
		results, err := utils.ProcessDirectory(
			ctx,
			inputPath,
			outputXML,
			*descriptionCount,
			&utils.RealAudioExtractor{},
			&utils.RealAudioTranscriber{},
			&utils.RealDescriptionGenerator{},
			evaluator,
		)
		if err != nil {
			log.Fatalf("Failed to process directory: %v", err)
		}
		fmt.Printf("Transcription results saved to %s\n", outputXML)
		fmt.Printf("Processed %d video(s)\n", len(results.Results))
	} else {
		// Process single file
		audioFile := filepath.Join(tmpDir, fmt.Sprintf("output_%d.wav", time.Now().Unix()))

		// Check if the audio file already exists
		if _, err := os.Stat(audioFile); err == nil {
			fmt.Printf("File '%s' already exists. Overwrite? [y/N] ", audioFile)
			var response string
			_, err := fmt.Scanln(&response)
			if err != nil {
				fmt.Printf("Error reading input: %v\n", err)
				return
			}
			if strings.ToLower(response) != "y" {
				fmt.Println("Not overwriting - exiting")
				return
			}
		}

		extractor := &utils.RealAudioExtractor{}
		hasAudio, err := extractor.ExtractAudio(ctx, inputPath, audioFile)
		if err != nil {
			log.Fatalf("Failed to extract audio: %v", err)
		}

		if !hasAudio {
			log.Fatalf("No audio found in the video file")
		}

		transcriber := &utils.RealAudioTranscriber{}
		transcription, err := transcriber.TranscribeAudio(ctx, audioFile, 5*time.Minute)
		if err != nil {
			log.Fatalf("Failed to transcribe audio: %v", err)
		}

		fmt.Println("Transcription:", transcription)

		descriptions, err := utils.GenerateDescriptions(transcription, filepath.Base(inputPath), *descriptionCount)
		if err != nil {
			log.Fatalf("Failed to generate descriptions: %v", err)
		}

		fmt.Println("Descriptions:")
		for i, desc := range descriptions {
			fmt.Printf("%d: %s\n", i+1, desc)
		}
	}

	// Clean up .tmp directory at exit
	cleanupTmpDir(tmpDir)
}

func cleanupTmpDir(tmpDir string) {
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		log.Printf("Failed to read .tmp directory: %v", err)
		return
	}

	for _, file := range files {
		err := os.Remove(filepath.Join(tmpDir, file.Name()))
		if err != nil {
			log.Printf("Failed to remove file %s: %v", file.Name(), err)
		}
	}
}
