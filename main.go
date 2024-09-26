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

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Create .tmp directory if it doesn't exist
	tmpDir := ".tmp"
	if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
		log.Fatalf("Failed to create .tmp directory: %v", err)
	}

	// Clean up .tmp directory at startup
	cleanupTmpDir(tmpDir)

	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <video_file_path>")
	}
	videoFile := os.Args[1]
	audioFile := filepath.Join(tmpDir, fmt.Sprintf("output_%d.wav", time.Now().Unix()))

	// Check if the audio file already exists
	if _, err := os.Stat(audioFile); err == nil {
		fmt.Printf("File '%s' already exists. Overwrite? [y/N] ", audioFile)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("Not overwriting - exiting")
			return
		}
	}

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

	err = extractAudio(ctx, videoFile, audioFile)
	if err != nil {
		log.Fatalf("Failed to extract audio: %v", err)
	}

	transcription, err := transcribeAudio(audioFile)
	if err != nil {
		log.Fatalf("Failed to transcribe audio: %v", err)
	}

	fmt.Println("Transcription:", transcription)

	description, err := GenerateDescription(transcription)
	if err != nil {
		log.Fatalf("Failed to generate description: %v", err)
	}

	fmt.Println("Description:", description)

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
