package types

import (
	"reflect"
	"testing"
)

func TestMergeAnnotations(t *testing.T) {
	tests := []struct {
		name      string
		preferred map[string]string
		aux       []string
		want      map[string]string
		wantErr   bool
	}{
		{
			name:      "Merge empty",
			preferred: nil,
			aux:       nil,
			want:      nil,
			wantErr:   false,
		},
		{
			name:      "Merge only preferred",
			preferred: map[string]string{"key1": "val1"},
			aux:       nil,
			want:      map[string]string{"key1": "val1"},
			wantErr:   false,
		},
		{
			name:      "Merge only aux",
			preferred: nil,
			aux:       []string{"key2=val2"},
			want:      map[string]string{"key2": "val2"},
			wantErr:   false,
		},
		{
			name:      "Merge both",
			preferred: map[string]string{"key1": "val1"},
			aux:       []string{"key2=val2"},
			want:      map[string]string{"key1": "val1", "key2": "val2"},
			wantErr:   false,
		},
		{
			name:      "Merge overwrite (aux overrides preferred)",
			preferred: map[string]string{"key1": "val1"},
			aux:       []string{"key1=val2"},
			want:      map[string]string{"key1": "val2"},
			wantErr:   false,
		},
		{
			name:      "Merge invalid aux",
			preferred: nil,
			aux:       []string{"invalid"},
			want:      nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MergeAnnotations(tt.preferred, tt.aux)
			if (err != nil) != tt.wantErr {
				t.Errorf("MergeAnnotations() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeAnnotations() = %v, want %v", got, tt.want)
			}
		})
	}
}
