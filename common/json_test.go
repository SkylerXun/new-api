package common

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJsonRawMessageToString(t *testing.T) {
	tests := []struct {
		name string
		data json.RawMessage
		want string
	}{
		{
			name: "object",
			data: json.RawMessage(`{"city":"Paris","days":0,"strict":false}`),
			want: `{"city":"Paris","days":0,"strict":false}`,
		},
		{
			name: "string",
			data: json.RawMessage(`"{\"city\":\"Paris\",\"days\":0,\"strict\":false}"`),
			want: `{"city":"Paris","days":0,"strict":false}`,
		},
		{
			name: "null",
			data: json.RawMessage(`null`),
			want: "",
		},
		{
			name: "empty",
			data: nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, JsonRawMessageToString(tt.data))
		})
	}
}

func TestSanitizeResponseModel(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{
			name: "json response",
			data: `{"id":"chatcmpl-1","model":"provider-secret","choices":[]}`,
			want: `{"choices":[],"id":"chatcmpl-1","model":"client-model"}`,
		},
		{
			name: "sse response",
			data: "data: {\"id\":\"chatcmpl-1\",\"model\":\"provider-secret\",\"choices\":[]}\n\n",
			want: "data: {\"choices\":[],\"id\":\"chatcmpl-1\",\"model\":\"client-model\"}\n\n",
		},
		{
			name: "nested response model",
			data: `{"response":{"model":"provider-secret"},"choices":[{"message":{"content":"provider-secret"}}]}`,
			want: `{"choices":[{"message":{"content":"provider-secret"}}],"response":{"model":"client-model"}}`,
		},
		{
			name: "model in content string is unchanged",
			data: `{"choices":[{"message":{"content":"{\"model\":\"provider-secret\"}"}}]}`,
			want: `{"choices":[{"message":{"content":"{\"model\":\"provider-secret\"}"}}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeResponseModel([]byte(tt.data), "client-model")
			require.Equal(t, tt.want, string(got))
		})
	}
}
