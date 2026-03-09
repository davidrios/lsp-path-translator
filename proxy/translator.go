package proxy

import (
	"strings"
)

type JSONPathTranslator struct {
	Source string
	Target string
}

func NewJSONPathTranslators(pathMap map[string]string) ([]JSONPathTranslator, error) {
	ret := []JSONPathTranslator{}
	for src, dest := range pathMap {
		ret = append(ret, JSONPathTranslator{Source: src, Target: dest})
	}

	return ret, nil
}

// Translate modifies the given JSON object (which is unmarshaled into an empty any)
// by recursively walking it and replacing string values that start with the Source path
// with the Target path.
func (t *JSONPathTranslator) Translate(v any) bool {
	if t.Source == "" || t.Target == "" || t.Source == t.Target {
		return false
	}
	t.translateNode(v)
	return true
}

func (t *JSONPathTranslator) Invert() {
	target := t.Target
	t.Target = t.Source
	t.Source = target
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
