package dockerfile

import (
	"reflect"
	"testing"
)

func TestNewFromDirective(t *testing.T) {
	testCases := []struct {
		name         string
		rawContent   string
		expectedFrom *FromDirective
	}{
		{
			name:       "with registry and tag",
			rawContent: "registry.example.com/myapp/my-image:v1.0",
			expectedFrom: &FromDirective{
				Type:      FROM,
				Content:   "registry.example.com/myapp/my-image:v1.0",
				Registry:  "registry.example.com",
				ImageName: "my-image",
				ImageTag:  "v1.0",
			},
		},
		{
			name:       "with registry and platform",
			rawContent: "registry.example.com/my-image@amd64",
			expectedFrom: &FromDirective{
				Type:      FROM,
				Content:   "registry.example.com/my-image@amd64",
				Registry:  "registry.example.com",
				ImageName: "my-image",
				Platform:  "amd64",
			},
		},
		{
			name:       "without registry, with tag",
			rawContent: "my-image:v2.0",
			expectedFrom: &FromDirective{
				Type:      FROM,
				Content:   "my-image:v2.0",
				ImageName: "my-image",
				ImageTag:  "v2.0",
			},
		},
		{
			name:       "without registry, with platform",
			rawContent: "my-image@arm64",
			expectedFrom: &FromDirective{
				Type:      FROM,
				Content:   "my-image@arm64",
				ImageName: "my-image",
				Platform:  "arm64",
			},
		},
		{
			name:       "without registry, tag, or platform",
			rawContent: "my-image",
			expectedFrom: &FromDirective{
				Type:      FROM,
				Content:   "my-image",
				ImageName: "my-image",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			from := NewFromDirective(tc.rawContent)
			if !reflect.DeepEqual(from, tc.expectedFrom) {
				t.Errorf("Expected %+v, got %+v", tc.expectedFrom, from)
			}
		})
	}
}
