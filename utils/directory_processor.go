package utils

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Description struct {
	Number  int    `xml:"number,attr"`
	Content string `xml:",chardata"`
}

type TranscriptionResult struct {
	VideoFile            string        `xml:"VideoFile"`
	AudioFile            string        `xml:"AudioFile"`
	Transcription        string        `xml:"Transcription"`
	Descriptions         []Description `xml:"Descriptions>Description"`
	BestDescriptionIndex int           `xml:"BestDescriptionIndex"`
}

type TranscriptionResults struct {
	XMLName xml.Name              `xml:"TranscriptionResults"`
	Results []TranscriptionResult `xml:"TranscriptionResult"`
}

var videoExtensions = map[string]bool{
	".mp4": true, ".mov": true, ".avi": true, ".mkv": true, ".wmv": true,
}

func ProcessDirectory(
	ctx context.Context,
	rootDir string,
	outputXML string,
	descriptionAttempts int,
	extractor AudioExtractor,
	transcriber AudioTranscriber,
	generator DescriptionGenerator,
	evaluator DescriptionEvaluator,
) (TranscriptionResults, error) {
	var results TranscriptionResults

	// Read existing XML file if it exists
	if _, err := os.Stat(outputXML); err == nil {
		file, err := os.Open(filepath.Clean(outputXML))
		if err != nil {
			return TranscriptionResults{}, fmt.Errorf("failed to open existing XML file: %v", err)
		}
		defer func() {
			if err := file.Close(); err != nil {
				fmt.Printf("Failed to close XML file: %v\n", err)
			}
		}()

		decoder := xml.NewDecoder(file)
		if err := decoder.Decode(&results); err != nil {
			return TranscriptionResults{}, fmt.Errorf("failed to decode existing XML: %v", err)
		}

		// Normalize paths in existing results
		for i := range results.Results {
			results.Results[i].VideoFile = filepath.ToSlash(filepath.Clean(results.Results[i].VideoFile))
			results.Results[i].AudioFile = filepath.ToSlash(filepath.Clean(results.Results[i].AudioFile))
		}
	}

	// Create a map of processed files using normalized paths
	processedFiles := make(map[string]*TranscriptionResult)
	for i, result := range results.Results {
		processedFiles[result.VideoFile] = &results.Results[i]
	}

	// Ensure .tmp directory exists
	if err := os.MkdirAll(".tmp", 0750); err != nil {
		return TranscriptionResults{}, fmt.Errorf("failed to create .tmp directory: %v", err)
	}

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			ext := strings.ToLower(filepath.Ext(d.Name()))
			if videoExtensions[ext] {
				// Compute relative path and normalize it
				relPath, err := filepath.Rel(rootDir, path)
				if err != nil {
					return fmt.Errorf("failed to compute relative path for '%s': %v", path, err)
				}
				normalizedPath := filepath.ToSlash(filepath.Clean(relPath))

				// Check if the file has been processed using normalized path
				existingResult, exists := processedFiles[normalizedPath]
				if exists && len(existingResult.Descriptions) >= descriptionAttempts {
					fmt.Printf("File '%s' already processed with sufficient descriptions. Skipping...\n", normalizedPath)
					return nil
				}

				// Process the video file (pass existing result if any)
				result, err := processVideoFile(ctx, path, normalizedPath, descriptionAttempts, extractor, transcriber, generator, evaluator, existingResult)
				if err != nil {
					return fmt.Errorf("failed to process video file '%s': %v", path, err)
				}

				if exists {
					// Update the existing result
					*existingResult = result
				} else {
					// Add new result
					results.Results = append(results.Results, result)
				}

				// Write the updated results to the XML file after each video is processed
				if err := writeXMLFile(outputXML, results); err != nil {
					return fmt.Errorf("failed to write XML file: %v", err)
				}
			}
		}
		return nil
	})

	if err != nil {
		return TranscriptionResults{}, err
	}

	return results, nil
}

func processVideoFile(
	ctx context.Context,
	videoFile string,
	relativePath string,
	descriptionAttempts int,
	extractor AudioExtractor,
	transcriber AudioTranscriber,
	generator DescriptionGenerator,
	evaluator DescriptionEvaluator,
	existingResult *TranscriptionResult,
) (TranscriptionResult, error) {
	var result TranscriptionResult

	// Use existing result if available
	if existingResult != nil {
		result = *existingResult
	} else {
		result.VideoFile = relativePath
	}

	// If there is no transcription, we need to extract audio and transcribe
	if result.Transcription == "" {
		// Generate a unique audio file name and normalize it
		audioFile := filepath.ToSlash(filepath.Clean(filepath.Join(".tmp", fmt.Sprintf("%s_%d.wav", strings.TrimSuffix(filepath.Base(relativePath), filepath.Ext(relativePath)), time.Now().UnixNano()))))

		// Use the injected extractor
		hasAudio, err := extractor.ExtractAudio(ctx, videoFile, audioFile)
		if err != nil {
			return TranscriptionResult{}, fmt.Errorf("failed to extract audio: %v", err)
		}
		if !hasAudio {
			fmt.Printf("Skipping file '%s' as it has no audio stream\n", relativePath)
			return TranscriptionResult{
				VideoFile: relativePath,
				AudioFile: "No audio",
			}, nil
		}

		result.AudioFile = audioFile

		// Use the injected transcriber
		transcription, err := transcriber.TranscribeAudio(ctx, audioFile, 5*time.Minute)
		if err != nil {
			return TranscriptionResult{}, fmt.Errorf("failed to transcribe audio: %v", err)
		}
		result.Transcription = transcription
	}

	// Calculate how many descriptions need to be generated
	existingDescriptionsCount := len(result.Descriptions)
	descriptionsToGenerate := descriptionAttempts - existingDescriptionsCount

	if descriptionsToGenerate > 0 {
		// Use the injected generator to generate missing descriptions
		newDescriptions, err := generator.GenerateDescriptions(result.Transcription, filepath.Base(relativePath), descriptionsToGenerate)
		if err != nil {
			return TranscriptionResult{}, fmt.Errorf("failed to generate descriptions: %v", err)
		}

		// Escape and append new descriptions
		for i, desc := range newDescriptions {
			escapedDesc := &bytes.Buffer{}
			if err := xml.EscapeText(escapedDesc, []byte(desc)); err != nil {
				return TranscriptionResult{}, fmt.Errorf("failed to escape description: %v", err)
			}
			result.Descriptions = append(result.Descriptions, Description{
				Number:  existingDescriptionsCount + i + 1,
				Content: escapedDesc.String(),
			})
		}

		// Re-evaluate descriptions
		bestIndex, err := evaluator.EvaluateDescriptions(getDescriptionContents(result.Descriptions), result.Transcription, filepath.Base(relativePath))
		if err != nil {
			return TranscriptionResult{}, fmt.Errorf("failed to evaluate descriptions: %v", err)
		}
		result.BestDescriptionIndex = bestIndex
		fmt.Printf("Suggested best description: %d\n", bestIndex)
	} else {
		fmt.Printf("Already have required number of descriptions for '%s'\n", relativePath)
	}

	return result, nil
}

func writeXMLFile(outputXML string, results TranscriptionResults) error {
	file, err := os.Create(filepath.Clean(outputXML))
	if err != nil {
		return fmt.Errorf("failed to create XML file '%s': %v", outputXML, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Failed to close XML file: %v\n", err)
		}
	}()

	encoder := xml.NewEncoder(file)
	encoder.Indent("", "  ")
	if err := encoder.Encode(results); err != nil {
		return fmt.Errorf("failed to encode XML to '%s': %v", outputXML, err)
	}

	// Make sure to flush the encoder
	if err := encoder.Flush(); err != nil {
		return fmt.Errorf("failed to flush XML encoder: %v", err)
	}

	fmt.Printf("XML results written to %s\n", outputXML)
	return nil
}

// Helper function to extract description contents
func getDescriptionContents(descriptions []Description) []string {
	contents := make([]string, len(descriptions))
	for i, desc := range descriptions {
		contents[i] = desc.Content
	}
	return contents
}
