package helper

import (
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"github.com/gin-gonic/gin"
)

// ModelHidingResponseWriter keeps channel model mappings private on the
// downstream side.  Adapters still send RelayInfo.UpstreamModelName upstream;
// this writer only rewrites response payloads as they leave the gateway.
//
// It is installed after RelayInfo has captured the client's original model, so
// all response paths (including provider-specific direct writes) get the same
// protection without requiring every adapter to remember to rewrite model
// fields independently.
type modelHidingResponseWriter struct {
	gin.ResponseWriter
	originalModel string
}

// NewModelHidingResponseWriter wraps a Gin response writer so JSON and SSE
// responses expose the model requested by the client rather than the mapped
// upstream model.  A blank model disables rewriting.
func NewModelHidingResponseWriter(writer gin.ResponseWriter, originalModel string) gin.ResponseWriter {
	if writer == nil || originalModel == "" {
		return writer
	}
	return &modelHidingResponseWriter{ResponseWriter: writer, originalModel: originalModel}
}

func (w *modelHidingResponseWriter) Write(data []byte) (int, error) {
	rewritten := common.SanitizeResponseModel(data, w.originalModel)
	n, err := w.ResponseWriter.Write(rewritten)
	if err != nil {
		return 0, err
	}
	if n != len(rewritten) {
		return 0, io.ErrShortWrite
	}
	// Report bytes consumed from the caller's input, not the size of the
	// rewritten payload (which may differ because model names have different
	// lengths).
	return len(data), nil
}

func (w *modelHidingResponseWriter) WriteString(data string) (int, error) {
	return w.Write([]byte(data))
}

// ReadFrom keeps io.Copy compatible with the wrapped writer while ensuring
// copied JSON/SSE payloads pass through the same sanitizer as ordinary writes.
func (w *modelHidingResponseWriter) ReadFrom(src io.Reader) (int64, error) {
	contentType := strings.ToLower(w.Header().Get("Content-Type"))
	if contentType != "" && !strings.Contains(contentType, "json") && !strings.Contains(contentType, "text/event-stream") {
		if readerFrom, ok := w.ResponseWriter.(io.ReaderFrom); ok {
			return readerFrom.ReadFrom(src)
		}
	}
	data, err := io.ReadAll(src)
	if err != nil {
		return 0, err
	}
	_, writeErr := w.Write(data)
	if writeErr != nil {
		return 0, writeErr
	}
	return int64(len(data)), nil
}

// Unwrap lets http.ResponseController reach optional capabilities of the
// underlying writer (for example, write deadlines used by stream relays).
func (w *modelHidingResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
