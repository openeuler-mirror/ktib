package parsingutils

import (
	"reflect"
	"testing"
)

func TestDockerfilePreprocessor_GetNormalizedContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "Basic Normalization",
			content:  "FROM ubuntu:latest\n\n# Comment\nRUN apt-get update\\\n    && apt-get install -y python3\n\nENV  PYTHON_VERSION=3.9.0",
			expected: "FROM ubuntu:latest\nRUN apt-get update && apt-get install -y python3\nENV PYTHON_VERSION=3.9.0",
		},
		{
			name:     "Env Substitution",
			content:  "FROM ubuntu:latest\nENV PYTHON_VERSION=3.9.0\nRUN pip install python-$PYTHON_VERSION",
			expected: "FROM ubuntu:latest\nENV PYTHON_VERSION=3.9.0\nRUN pip install python-3.9.0",
		},
		{
			name:     "Env Substitution with Braces",
			content:  "FROM ubuntu:latest\nENV PYTHON_VERSION=3.9.0\nRUN pip install python-${PYTHON_VERSION}",
			expected: "FROM ubuntu:latest\nENV PYTHON_VERSION=3.9.0\nRUN pip install python-3.9.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewDockerfilePreprocessor(tt.content)
			result := p.GetNormalizedContent()
			if result != tt.expected {
				t.Errorf("GetNormalizedContent() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestDockerfilePreprocessor_getEnvBasic(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected map[string]string
	}{
		{
			name:    "Basic ENV variables",
			content: "FROM ubuntu:latest\nENV PYTHON_VERSION=3.9.0 DEBIAN_FRONTEND=noninteractive",
			expected: map[string]string{
				"PYTHON_VERSION":  "3.9.0",
				"DEBIAN_FRONTEND": "noninteractive",
			},
		},
		{
			name:    "ENV with quotes",
			content: "FROM ubuntu:latest\nENV PYTHON_VERSION=\"3.9.0\" DEBIAN_FRONTEND='noninteractive'",
			expected: map[string]string{
				"PYTHON_VERSION":  "3.9.0",
				"DEBIAN_FRONTEND": "noninteractive",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewDockerfilePreprocessor(tt.content)
			result := p.getEnvBasic()
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("getEnvBasic() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestDockerfilePreprocessor_getEnvKeyValue(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected map[string]string
	}{
		{
			name:    "ENV with key-value pairs",
			content: "FROM ubuntu:latest\nENV PYTHON_VERSION=3.9.0 DEBIAN_FRONTEND=noninteractive APP_HOME=/app",
			expected: map[string]string{
				"PYTHON_VERSION":  "3.9.0",
				"DEBIAN_FRONTEND": "noninteractive",
				"APP_HOME":        "/app",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewDockerfilePreprocessor(tt.content)
			result := p.getEnvKeyValue()
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("getEnvKeyValue() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
