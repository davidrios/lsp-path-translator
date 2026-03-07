package proxy

import (
	"strings"
)

type JSONPathTranslator struct {
	Source string
	Target string
}

func NewJSONPathTranslator(source, target string) *JSONPathTranslator {
	return &JSONPathTranslator{
		Source: source,
		Target: target,
	}
}

// Translate modifies the given JSON object (which is unmarshaled into an empty any)
// by recursively walking it and replacing string values that start with the Source path
// with the Target path.
func (t *JSONPathTranslator) Translate(v any) {
	if t.Source == "" || t.Target == "" || t.Source == t.Target {
		return
	}
	t.translateNode(v)
}

func (t *JSONPathTranslator) translateNode(v any) {
	switch node := v.(type) {
	case map[string]any:
		for key, val := range node {
			if strVal, ok := val.(string); ok {
				node[key] = t.translateString(strVal)
			} else {
				t.translateNode(val)
			}
		}
	case []any:
		for i, val := range node {
			if strVal, ok := val.(string); ok {
				node[i] = t.translateString(strVal)
			} else {
				t.translateNode(val)
			}
		}
	}
}

func (t *JSONPathTranslator) translateString(s string) string {
	if after, ok := strings.CutPrefix(s, t.Source); ok {
		return t.Target + after
	}

	sourceURI := "file://" + t.Source
	if after, ok := strings.CutPrefix(s, sourceURI); ok {
		return "file://" + t.Target + after
	}

	return s
}
