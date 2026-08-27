package types

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCloneWithMessageRewritesStructuredPayloadWithoutMutatingOriginal(t *testing.T) {
	t.Parallel()

	t.Run("OpenAI", func(t *testing.T) {
		t.Parallel()
		original := WithOpenAIError(OpenAIError{
			Message: "raw upstream message",
			Type:    "server_error",
			Code:    "server_error",
		}, http.StatusServiceUnavailable)

		clone := original.CloneWithMessage("Service is busy")

		require.NotSame(t, original, clone)
		require.Equal(t, "raw upstream message", original.Error())
		require.Equal(t, "raw upstream message", original.ToOpenAIError().Message)
		require.Equal(t, "Service is busy", clone.Error())
		require.Equal(t, "Service is busy", clone.ToOpenAIError().Message)
		require.Equal(t, original.StatusCode, clone.StatusCode)
		require.Equal(t, original.GetErrorCode(), clone.GetErrorCode())
	})

	t.Run("Claude", func(t *testing.T) {
		t.Parallel()
		original := WithClaudeError(ClaudeError{
			Message: "raw upstream message",
			Type:    "overloaded_error",
		}, http.StatusServiceUnavailable)

		clone := original.CloneWithMessage("Service is busy")

		require.Equal(t, "raw upstream message", original.ToClaudeError().Message)
		require.Equal(t, "Service is busy", clone.ToClaudeError().Message)
	})

	t.Run("unstructured", func(t *testing.T) {
		t.Parallel()
		original := NewError(errors.New("raw internal message"), ErrorCodeDoRequestFailed)

		clone := original.CloneWithMessage("Upstream request failed")

		require.Equal(t, "raw internal message", original.Error())
		require.Equal(t, "Upstream request failed", clone.ToOpenAIError().Message)
	})

	require.Nil(t, (*NewAPIError)(nil).CloneWithMessage("unused"))
}
