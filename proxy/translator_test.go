package proxy

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestJSONPathTranslator_Translate(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		target   string
		input    string
		expected string
	}{
		{
			name:     "simple path translation in map",
			source:   "/src/code",
			target:   "/dst/code",
			input:    `{"path": "/src/code/file.go"}`,
			expected: `{"path": "/dst/code/file.go"}`,
		},
		{
			name:     "uri translation in map",
			source:   "/src/code",
			target:   "/dst/code",
			input:    `{"uri": "file:///src/code/file.go"}`,
			expected: `{"uri": "file:///dst/code/file.go"}`,
		},
		{
			name:   "deeply nested translation",
			source: "/src/code",
			target: "/dst/code",
			input: `{
				"params": {
					"settings": {
						"workspace": "/src/code"
					},
					"files": [
						"/src/code/a.go",
						"file:///src/code/b.go"
					]
				}
			}`,
			expected: `{
				"params": {
					"settings": {
						"workspace": "/dst/code"
					},
					"files": [
						"/dst/code/a.go",
						"file:///dst/code/b.go"
					]
				}
			}`,
		},
		{
			name:     "no match",
			source:   "/src/code",
			target:   "/dst/code",
			input:    `{"path": "/other/code/file.go"}`,
			expected: `{"path": "/other/code/file.go"}`,
		},
		{
			name:     "partial match not changed (must be prefix)",
			source:   "/src/code",
			target:   "/dst/code",
			input:    `{"path": "/foo/src/code/file.go"}`,
			expected: `{"path": "/foo/src/code/file.go"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var inputData any
			if err := json.Unmarshal([]byte(tt.input), &inputData); err != nil {
				t.Fatalf("failed to unmarshal input: %v", err)
			}

			translators, err := NewJSONPathTranslators([]string{tt.source + "::" + tt.target})
			if err != nil {
				t.Fatalf("Failed to create translators: %v", err)
			}
			translators[0].Translate(inputData)

			var expectedData any
			if err := json.Unmarshal([]byte(tt.expected), &expectedData); err != nil {
				t.Fatalf("failed to unmarshal expected: %v", err)
			}

			if !reflect.DeepEqual(inputData, expectedData) {
				t.Errorf("Translate() got = %v, want %v", inputData, expectedData)
			}
		})
	}
}
