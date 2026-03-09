package proxy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
)

type Stream struct {
	reader      *bufio.Reader
	writer      io.Writer
	translators *[]JSONPathTranslator
	logMessages bool
}

func NewStream(rw io.ReadWriter, translators *[]JSONPathTranslator) *Stream {
	return &Stream{
		reader:      bufio.NewReader(rw),
		writer:      rw,
		translators: translators,
	}
}

func NewStreamRW(r io.Reader, w io.Writer, translators *[]JSONPathTranslator, logMessages bool) *Stream {
	return &Stream{
		reader:      bufio.NewReader(r),
		writer:      w,
		translators: translators,
		logMessages: logMessages,
	}
}

// ReadAndTranslate reads the next LSP message from the stream, parses it into JSON,
// translates the paths, marshals it back to JSON, and returns the modified payload.
// Returns an error if the connection is closed or reading fails.
func (s *Stream) ReadAndTranslate() ([]byte, error) {
	// Read headers
	contentLength := -1
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimRight(line, "\r\n")

		if line == "" {
			// End of headers
			break
		}

		if after, ok := strings.CutPrefix(line, "Content-Length: "); ok {
			lengthStr := after
			l, err := strconv.Atoi(lengthStr)
			if err != nil {
				return nil, fmt.Errorf("invalid Content-Length: %w", err)
			}
			contentLength = l
		}
	}

	if contentLength == -1 {
		return nil, fmt.Errorf("missing Content-Length header")
	}

	// Read body
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(s.reader, body); err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	// Try to parse the body as JSON
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		// If it's not JSON, we just return the raw body instead of failing.
		return body, nil
	}

	count := 0
	for _, translator := range *s.translators {
		if translator.Translate(payload) {
			count += 1
		}
	}

	if count == 0 {
		return body, nil
	}

	// Marshal back to JSON
	translatedBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal translated payload: %w", err)
	}

	if s.logMessages {
		log.Printf("%s -> %s\n", body, translatedBody)
	}

	return translatedBody, nil
}

// Write writes the given payload as an LSP message (with headers).
func (s *Stream) Write(payload []byte) error {
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(payload))

	if _, err := s.writer.Write([]byte(header)); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	if _, err := s.writer.Write(payload); err != nil {
		return fmt.Errorf("failed to write payload: %w", err)
	}

	return nil
}
