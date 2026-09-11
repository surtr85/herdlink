package herdr

import (
	"strings"
	"testing"
)

func TestStripAnsi(t *testing.T) {
	colored := "\x1b[31mError:\x1b[0m \x1b[1;32mServer is running\x1b[0m\n"
	stripped := StripAnsi(colored)
	expected := "Error: Server is running\n"
	if stripped != expected {
		t.Errorf("expected %q, got %q", expected, stripped)
	}
}

func TestSmartTruncate(t *testing.T) {
	var lines []string
	for i := 1; i <= 100; i++ {
		lines = append(lines, strings.Repeat("x", 10))
	}
	content := strings.Join(lines, "\n")

	truncated := SmartTruncate(content, 30)
	if !strings.Contains(truncated, "lines truncated for token efficiency") {
		t.Errorf("expected truncation marker in output")
	}

	short := "line 1\nline 2\nline 3\n"
	if notTruncated := SmartTruncate(short, 10); notTruncated != short {
		t.Errorf("expected %q, got %q", short, notTruncated)
	}
}
