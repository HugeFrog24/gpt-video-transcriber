package utils

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
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

func ProcessDirectory(
	ctx context.Context,
	dir string,
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
		file, err := os.Open(outputXML)
		if err != nil {
			return TranscriptionResults{}, fmt.Errorf("failed to open existing XML file: %v", err)
		}
		defer file.Close()

		decoder := xml.NewDecoder(file)
		if err := decoder.Decode(&results); err != nil {
			return TranscriptionResults{}, fmt.Errorf("failed to decode existing XML: %v", err)
		}
	}

	// Create a map of processed files to their results
	processedFiles := make(map[string]*TranscriptionResult)
	for i, result := range results.Results {
		processedFiles[result.VideoFile] = &results.Results[i]
	}

	// Ensure .tmp directory exists
	if err := os.MkdirAll(".tmp", os.ModePerm); err != nil {
		return TranscriptionResults{}, fmt.Errorf("failed to create .tmp directory: %v", err)
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".mp4") {
			// Check if the file has been processed
			existingResult, exists := processedFiles[path]
			if exists && len(existingResult.Descriptions) >= descriptionAttempts {
				fmt.Printf("File '%s' already processed with sufficient descriptions. Skipping...\n", path)
				return nil
			}

			// Process the video file (pass existing result if any)
			result, err := processVideoFile(ctx, path, descriptionAttempts, extractor, transcriber, generator, evaluator, existingResult)
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
		result.VideoFile = videoFile
	}

	// If there is no transcription, we need to extract audio and transcribe
	if result.Transcription == "" {
		// Generate a unique audio file name
		audioFile := filepath.Join(".tmp", fmt.Sprintf("%s_%d.wav", strings.TrimSuffix(filepath.Base(videoFile), filepath.Ext(videoFile)), time.Now().UnixNano()))

		// Use the injected extractor
		hasAudio, err := extractor.ExtractAudio(ctx, videoFile, audioFile)
		if err != nil {
			return TranscriptionResult{}, fmt.Errorf("failed to extract audio: %v", err)
		}
		if !hasAudio {
			fmt.Printf("Skipping file '%s' as it has no audio stream\n", videoFile)
			return TranscriptionResult{
				VideoFile: videoFile,
				AudioFile: "No audio",
			}, nil
		}

		result.AudioFile = audioFile

		// Use the injected transcriber
		transcription, err := transcriber.TranscribeAudio(audioFile, 5*time.Minute)
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
		newDescriptions, err := generator.GenerateDescriptions(result.Transcription, filepath.Base(videoFile), descriptionsToGenerate)
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
		bestIndex, err := evaluator.EvaluateDescriptions(getDescriptionContents(result.Descriptions), result.Transcription, filepath.Base(videoFile))
		if err != nil {
			return TranscriptionResult{}, fmt.Errorf("failed to evaluate descriptions: %v", err)
		}
		result.BestDescriptionIndex = bestIndex
		fmt.Printf("Suggested best description: %d\n", bestIndex)
	} else {
		fmt.Printf("Already have required number of descriptions for '%s'\n", videoFile)
	}

	return result, nil
}

func writeXMLFile(outputXML string, results TranscriptionResults) error {
	file, err := os.Create(outputXML)
	if err != nil {
		return fmt.Errorf("failed to create XML file '%s': %v", outputXML, err)
	}
	defer file.Close()

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
