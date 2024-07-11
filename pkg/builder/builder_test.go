package builder

import (
	"regexp"
	"testing"
)

func TestStripComments(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "empty input",
			input:    []byte(""),
			expected: "",
		},
		{
			name:     "no comment line",
			input:    []byte("line1\nline2"),
			expected: "line1\nline2",
		},
		{
			name:     "with comment line",
			input:    []byte("#hashbutnotacomment\n#alsonotacomment\nline3"),
			expected: "line3",
		},
		{
			name:     "comments and empty lines",
			input:    []byte("line1\n#comment\n\nline3\n#another comment"),
			expected: "line1\nline3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripComments(tt.input); got != tt.expected {
				t.Errorf("stripComments() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLineContinuationRegex(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no comments",
			input:    "FROM golang:1.20\nRUN echo Hello, World!",
			expected: "FROM golang:1.20\nRUN echo Hello, World!",
		},
		{
			name:     "with comments",
			input:    "# This is a comment\nFROM golang:1.20\n# Another comment\nRUN echo Hello, World!",
			expected: "FROM golang:1.20\nRUN echo Hello, World!",
		},
		{
			name:     "with_line_continuation_in_FROM",
			input:    "FROM \\\ngolang:1.20\nRUN echo Hello, World!",
			expected: "FROM golang:1.20\nRUN echo Hello, World!",
		},
		{
			name:     "with_line_continuation_in_RUN",
			input:    "FROM golang:1.20\nRUN echo Hello, \\\nWorld!",
			expected: "FROM golang:1.20\nRUN echo Hello, World!",
		},
		{
			name:     "with_line_continuation_and_comments",
			input:    "# Preamble comment\nFROM \\\n# This is a comment\ngolang:1.20\nRUN echo Hello, World!",
			expected: "FROM golang:1.20\nRUN echo Hello, World!",
		},
		{
			name:     "with_line_continuation_and_whitespace",
			input:    "FROM \\\t\n\t\t\ngolang:1.20\nRUN echo Hello, World!",
			expected: "FROM golang:1.20\nRUN echo Hello, World!",
		},
	}
	lineContinuation := regexp.MustCompile(`\\\s*\n`)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := lineContinuation.ReplaceAllString(stripComments([]byte(tt.input)), "")
			if actual != tt.expected {
				t.Errorf("stripComments() = %v, want %v", actual, tt.expected)
			}
		})
	}
}
