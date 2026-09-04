package utils

import "testing"

func TestCheckMediaPathRejectsUnsafePaths(t *testing.T) {
	unsafe := []string{
		"",
		"-y",
		"-i",
		"http://example.com/evil.mp4",
		"concat:a.wav|b.wav",
		"subfile,,start,0,end,0,,:/etc/passwd",
		"my:clip.mp4",
	}
	for _, path := range unsafe {
		if err := checkMediaPath("audio", path); err == nil {
			t.Errorf("checkMediaPath(%q) = nil, want error", path)
		}
	}
}

func TestCheckMediaPathAllowsRealPaths(t *testing.T) {
	safe := []string{
		"/home/user/videos/clip.mp4",
		".tmp/clip_1730000000.wav",
		`C:\Users\admin\videos\clip.mp4`,
		"clip-with-dashes.mp4",
		"movie.2024.mp4",
		// exec.Command uses no shell, so these are ordinary file names.
		"weird name; rm -rf /.mp4",
		"$(whoami).mp4",
	}
	for _, path := range safe {
		if err := checkMediaPath("video", path); err != nil {
			t.Errorf("checkMediaPath(%q) = %v, want nil", path, err)
		}
	}
}
