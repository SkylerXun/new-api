package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOaiStreamHandlerMapsFinalNewAPIStreamDisconnectChunk(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })

	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	common.SetContextKey(c, constant.ContextKeyChannelErrorMessageMapping,
		`{"stream_disconnect":"服务繁忙，请稍后重试"}`)

	upstreamMessage := "stream disconnected before completion: upstream private detail"
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(
			`data: {"error":{"type":"upstream_error","message":"` + upstreamMessage + `"}}` + "\n",
		)),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeNewAPI,
			UpstreamModelName: "gpt-test",
		},
		RelayMode:   relayconstant.RelayModeChatCompletions,
		RelayFormat: types.RelayFormatOpenAI,
		IsStream:    true,
		DisablePing: true,
	}

	usage, relayErr := OaiStreamHandler(c, info, resp)
	require.Nil(t, relayErr)
	require.NotNil(t, usage)
	require.Zero(t, usage.TotalTokens)
	require.Contains(t, recorder.Body.String(), "服务繁忙，请稍后重试")
	require.NotContains(t, recorder.Body.String(), "stream disconnected before completion")
	require.NotContains(t, recorder.Body.String(), "upstream private detail")
}

func TestSanitizeNewAPIPlainTextStreamDisconnect(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(c, constant.ContextKeyChannelErrorMessageMapping,
		`{"stream_disconnect":"服务繁忙，请稍后重试"}`)
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeNewAPI}}

	result := sanitizeNewAPIStreamErrorData(c, info,
		"stream disconnected before completion: upstream private detail")
	require.Contains(t, result, "服务繁忙，请稍后重试")
	require.NotContains(t, result, "stream disconnected before completion")
	require.NotContains(t, result, "upstream private detail")
}
