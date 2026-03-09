package proxy

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestStream_ReadAndTranslate(t *testing.T) {
	// Input JSON with source path
	inputJSON := `{"path":"/src/workspace/file.go"}`
	// Header + Body
	inputPayload := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(inputJSON), inputJSON)

	reader := strings.NewReader(inputPayload)
	var writer bytes.Buffer

	translators, err := NewJSONPathTranslators(map[string]string{"/src/workspace": "/dst/workspace"})
	if err != nil {
		t.Fatalf("Failed to create translators: %v", err)
	}
	stream := NewStreamRW(reader, &writer, &translators, false)

	translatedBytes, err := stream.ReadAndTranslate()
	if err != nil {
		t.Fatalf("ReadAndTranslate() failed: %v", err)
	}

	expectedOutput := `{"path":"/dst/workspace/file.go"}`
	if string(translatedBytes) != expectedOutput {
		t.Errorf("Expected translated payload %q, got %q", expectedOutput, string(translatedBytes))
	}
}

func TestStream_Write(t *testing.T) {
	var writer bytes.Buffer
	translators, err := NewJSONPathTranslators(map[string]string{})
	if err != nil {
		t.Fatalf("Failed to create translators: %v", err)
	}
	stream := NewStreamRW(&bytes.Buffer{}, &writer, &translators, false)

	payload := []byte(`{"result":"ok"}`)
	err = stream.Write(payload)
	if err != nil {
		t.Fatalf("Write() failed: %v", err)
	}

	expectedOutput := "Content-Length: 15\r\n\r\n{\"result\":\"ok\"}"
	if writer.String() != expectedOutput {
		t.Errorf("Expected output %q, got %q", expectedOutput, writer.String())
	}
}
