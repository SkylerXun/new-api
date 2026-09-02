package common

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
)

func Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func UnmarshalJsonStr(data string, v any) error {
	return json.Unmarshal(StringToByteSlice(data), v)
}

func DecodeJson(reader io.Reader, v any) error {
	return json.NewDecoder(reader).Decode(v)
}

func Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func GetJsonType(data json.RawMessage) string {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return "unknown"
	}
	firstChar := trimmed[0]
	switch firstChar {
	case '{':
		return "object"
	case '[':
		return "array"
	case '"':
		return "string"
	case 't', 'f':
		return "boolean"
	case 'n':
		return "null"
	default:
		return "number"
	}
}

// SanitizeResponseModel replaces model fields in a JSON response (or in each
// SSE data event) with the model originally requested by the
// client.  Relay adapters must continue using the mapped model for upstream
// requests, but the mapped name is an internal routing detail and must not be
// exposed in downstream responses.
//
// Non-JSON payloads and events without a model field are returned unchanged.
// Model names embedded in content or tool-argument strings are not modified;
// JSON object fields named "model" are rewritten recursively so formats such
// as Responses (response.model) and Claude (message.model) are covered too.
func SanitizeResponseModel(data []byte, originalModel string) []byte {
	if len(data) == 0 || originalModel == "" {
		return data
	}

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		return sanitizeJSONModels(data, originalModel)
	}

	// Streaming helpers write complete SSE events, but an event may contain
	// multiple lines. Preserve the framing and rewrite only data: JSON lines.
	if bytes.Contains(data, []byte("data:")) {
		changed := false
		lines := strings.SplitAfter(string(data), "\n")
		for i, line := range lines {
			lineStart := strings.Index(line, "data:")
			if lineStart < 0 {
				continue
			}
			payloadStart := lineStart + len("data:")
			payload := line[payloadStart:]
			leading := len(payload) - len(strings.TrimLeft(payload, " \t"))
			payloadJSON := strings.TrimSpace(payload)
			if payloadJSON == "" || payloadJSON == "[DONE]" || !strings.HasPrefix(payloadJSON, "{") {
				continue
			}
			rewritten := sanitizeJSONModels([]byte(payloadJSON), originalModel)
			if string(rewritten) == payloadJSON {
				continue
			}
			// Keep the original indentation/prefix and line ending.
			ending := ""
			if strings.HasSuffix(payload, "\r\n") {
				ending = "\r\n"
			} else if strings.HasSuffix(payload, "\n") {
				ending = "\n"
			}
			lines[i] = line[:payloadStart] + payload[:leading] + string(rewritten) + ending
			changed = true
		}
		if changed {
			return []byte(strings.Join(lines, ""))
		}
	}

	return data
}

func sanitizeJSONModels(data []byte, originalModel string) []byte {
	var object map[string]json.RawMessage
	modelJSON, err := Marshal(originalModel)
	if err != nil {
		return data
	}
	if err := Unmarshal(data, &object); err == nil {
		changed := false
		for key, value := range object {
			if key == "model" {
				object[key] = modelJSON
				changed = true
				continue
			}
			rewrittenValue := sanitizeJSONModels(value, originalModel)
			if !bytes.Equal(rewrittenValue, value) {
				object[key] = rewrittenValue
				changed = true
			}
		}
		if !changed {
			return data
		}
		rewritten, marshalErr := Marshal(object)
		if marshalErr == nil {
			return rewritten
		}
		return data
	}

	var array []json.RawMessage
	if err := Unmarshal(data, &array); err != nil {
		return data
	}
	changed := false
	for i, value := range array {
		rewrittenValue := sanitizeJSONModels(value, originalModel)
		if !bytes.Equal(rewrittenValue, value) {
			array[i] = rewrittenValue
			changed = true
		}
	}
	if !changed {
		return data
	}
	rewritten, err := Marshal(array)
	if err != nil {
		return data
	}
	return rewritten
}

// JsonRawMessageToString returns JSON strings as their decoded value and other JSON values as raw text.
func JsonRawMessageToString(data json.RawMessage) string {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return ""
	}
	if trimmed[0] != '"' {
		return string(trimmed)
	}
	var value string
	if err := Unmarshal(trimmed, &value); err != nil {
		return string(trimmed)
	}
	return value
}
