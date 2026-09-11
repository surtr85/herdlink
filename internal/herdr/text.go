package herdr

import (
	"fmt"
	"regexp"
	"strings"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\].*?(\x07|\x1b\\)|\x1b[()][AB012]|\x1b\[\?[0-9]+[hl]`)

// StripAnsi removes ANSI escape codes and terminal control sequences from text
func StripAnsi(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// SmartTruncate truncates output to maxLines preserving head and tail with a marker banner
func SmartTruncate(s string, maxLines int) string {
	if s == "" || maxLines <= 0 {
		return s
	}

	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	totalLines := len(lines)
	if totalLines <= maxLines {
		return s
	}

	headCount := maxLines / 3
	if headCount < 5 {
		headCount = 5
	}
	tailCount := maxLines - headCount
	if tailCount < 5 {
		tailCount = 5
	}

	if headCount+tailCount >= totalLines {
		return s
	}

	truncatedCount := totalLines - headCount - tailCount

	head := strings.Join(lines[:headCount], "\n")
	tail := strings.Join(lines[totalLines-tailCount:], "\n")
	marker := fmt.Sprintf("\n... [%d lines truncated for token efficiency] ...\n", truncatedCount)

	return head + marker + tail + "\n"
}
