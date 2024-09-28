# Video Transcription and Description Generator

## What This Project Does

This tool helps content creators and video enthusiasts recover lost metadata for their video files. It automates transcribing video content and generating descriptive summaries, making it easier to organize and understand your video library.

## Key features:

1. Audio Extraction:
   - Extracts audio from video files for transcription.
2. Transcription:
   - Converts extracted audio into text.
3. Language Detection:
   - Identifies the language of the transcribed text.
4. Text Summarization:
   - Summarizes long transcriptions to fit within API limits.
5. Description Generation:
   - Uses AI to create multiple descriptive summaries based on the transcription.
6. Description Evaluation:
   - Selects the best description from generated options.

## Why It Exists

This project addresses the common problem of losing valuable metadata. Whether due to technical issues, platform changes, or accidental deletions, losing video descriptions can be frustrating and time-consuming to recreate.

By automating transcription and description generation, this tool aims to:

1. Save Time
2. Recover Lost Information
3. Improve Accessibility
4. Enhance Searchability

## Inspiration

The project was inspired by the experience of losing original video metadata. Many creators have faced situations where they've lost access to their original descriptions, tags, and other important information due to platform changes, account suspensions, or data loss.

This solution aims to alleviate the stress and time investment required to manually recreate lost metadata, serving as a helpful tool for content creators, archivists, and anyone who values organized and accessible video content.

## Project Status

This project is in development and not production-ready. The following improvements are necessary:

- [ ] **Configurable AI Prompt**: The AI prompt is hardcoded with a specific channel name (HugeFrog24). It needs to be configurable for different users and use cases.

- [ ] **Improved Error Handling**: Error handling for large texts needs enhancement to ensure robustness.

## How to Use

### For Developers

1. **Prerequisites:**
   - Go installed
   - FFmpeg available in system PATH
   - OpenAI API key

2. **Installation:**
   - Clone the repository
   - Run `go mod tidy` to install dependencies

3. **Configuration:**
   - Create a `.env` file in the project root
   - Add your OpenAI API key:
     ```
     OPENAI_API_KEY=your_api_key_here
     ```

4. **Usage:**
   - Process a single video:
     ```
     go run main.go "path/to/video.mp4"
     ```
   - Process a directory of videos:
     ```
     go run main.go "path/to/video/directory"
     ```
   - Specify number of descriptions (default: 3):
     ```
     go run main.go -descriptions 5 "path/to/video.mp4"
     ```

5. **Output:**
   - Single file: transcription and descriptions printed to console
   - Directory: results saved in `transcription_results.xml`

6. **Cleanup:**
   - Temporary files are automatically removed after processing

### For End Users

1. **Download:**
   - Get the pre-built executable for your OS from the releases page.

2. **Configuration:**
   - Create a `.env` file in the executable's directory
   - Add your OpenAI API key:
     ```
     OPENAI_API_KEY=your_api_key_here
     ```

3. **Usage:**

   - **Windows:**
     - Open Command Prompt
     - Navigate to the executable's directory
     - Process a single video:
       ```
       transcription_tool.exe "path\to\video.mp4"
       ```
     - Process a directory of videos:
       ```
       transcription_tool.exe "path\to\video\directory"
       ```

   - **Mac/Linux:**
     - Open Terminal
     - Navigate to the executable's directory
     - Make executable runnable (once):
       ```
       chmod +x transcription_tool
       ```
     - Process a single video:
       ```
       ./transcription_tool "path/to/video.mp4"
       ```
     - Process a directory of videos:
       ```
       ./transcription_tool "path/to/video/directory"
       ```

4. **Output:**
   - Single file: transcription and descriptions printed to console
   - Directory: results saved in `transcription_results.xml`

5. **Cleanup:**
   - Temporary files are automatically removed after processing

Note: This tool requires an active internet connection to use the OpenAI API for transcription and description generation.

While this tool is powerful, regularly backing up your original metadata is recommended to prevent future loss.