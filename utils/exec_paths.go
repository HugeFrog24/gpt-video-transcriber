package utils

import (
	"fmt"
	"strings"
)

// checkMediaPath rejects paths that ffmpeg or ffprobe would treat as something
// other than a plain file. Every current caller passes an absolute path or a
// generated ".tmp/" name, so this enforces an existing invariant rather than
// fixing a live bug, but it keeps a future caller from turning a file name into
// an option or a URL. Shell metacharacters are deliberately not checked:
// exec.Command passes argv straight to the OS, so no shell ever sees them.
//
// Note that a relative path whose first component contains a colon
// ("my:clip.mp4") is rejected, because ffmpeg would read it as a protocol too.
func checkMediaPath(role, path string) error {
	if path == "" {
		return fmt.Errorf("empty %s path", role)
	}
	if strings.HasPrefix(path, "-") {
		return fmt.Errorf("invalid %s path %q: a leading '-' is parsed as an option", role, path)
	}
	if looksLikeURLProtocol(path) {
		return fmt.Errorf("invalid %s path %q: resolves as a URL protocol, not a file", role, path)
	}
	return nil
}

// looksLikeURLProtocol reports whether ffmpeg would open path through a protocol
// handler ("http:", "concat:", "subfile,...:") instead of as a file. It mirrors
// libavformat's url_find_protocol, including its exemption for a single-letter
// Windows drive prefix.
func looksLikeURLProtocol(path string) bool {
	n := 0
	for n < len(path) && isURLSchemeChar(path[n]) {
		n++
	}
	if n == 0 || n >= len(path) {
		return false
	}
	if n == 1 && path[n] == ':' {
		return false // Windows drive letter, e.g. "C:\videos\clip.mp4"
	}
	if path[n] == ':' {
		return true
	}
	// "subfile,<options>:<path>" style: a comma in the scheme, colon further on.
	return path[n] == ',' && strings.ContainsRune(path[n+1:], ':')
}

// isURLSchemeChar matches the characters ffmpeg allows in a protocol name
// (libavformat's URL_SCHEME_CHARS).
func isURLSchemeChar(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	case c == '+', c == '-', c == '.':
		return true
	}
	return false
}
