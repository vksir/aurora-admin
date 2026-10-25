package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"aurora-admin/pkg/log"
)

// captureFormatter 捕获格式化参数，不真正输出日志。
func captureFormatter(captured *[]LogFormatterParams) LogFormatter {
	return func(params LogFormatterParams, logger *log.Logger) {
		*captured = append(*captured, params)
	}
}

func newTestEngine(t *testing.T, captured *[]LogFormatterParams) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	e := gin.New()
	e.Use(Logger(LoggerConfig{Formatter: captureFormatter(captured)}, log.DefaultLogger()))
	return e
}

// POST 请求：请求体被快照进日志，且 handler 仍能读到原始 body。
func TestLoggerRequestBodySnapshot(t *testing.T) {
	var captured []LogFormatterParams
	e := newTestEngine(t, &captured)

	var handlerSeen string
	e.POST("/echo", func(c *gin.Context) {
		buf := &bytes.Buffer{}
		_, _ = buf.ReadFrom(c.Request.Body)
		handlerSeen = buf.String()
		c.String(http.StatusOK, "ok")
	})

	body := `{"hello":"world"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(body))
	e.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, body, handlerSeen, "handler 应能读取原始请求体")
	assert.Len(t, captured, 1)
	assert.Equal(t, body, captured[0].RequestBody)
}

// GET 请求：默认 skipper 跳过，不记录日志、不产生任何缓冲开销。
func TestLoggerSkipsGet(t *testing.T) {
	var captured []LogFormatterParams
	e := newTestEngine(t, &captured)

	e.GET("/download", func(c *gin.Context) {
		c.String(http.StatusOK, strings.Repeat("x", 200_000))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/download", nil)
	e.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 200_000, w.Body.Len(), "响应应完整透传")
	assert.Empty(t, captured, "GET 请求不应记录日志")
}

// 大响应：透传完整内容，但日志缓冲被限制在 maxLoggedBodySize 内。
func TestLoggerLargeResponseTruncated(t *testing.T) {
	var captured []LogFormatterParams
	e := newTestEngine(t, &captured)

	e.POST("/download", func(c *gin.Context) {
		c.String(http.StatusOK, strings.Repeat("x", 200_000))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/download", nil)
	e.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 200_000, w.Body.Len(), "响应应完整透传给客户端")
	assert.Len(t, captured, 1)
	assert.Len(t, captured[0].ResponseBody.String(), maxLoggedBodySize, "日志缓冲应有上限")
}

// 大请求体：不读入内存，直接跳过快照。
func TestLoggerLargeRequestSkipped(t *testing.T) {
	var captured []LogFormatterParams
	e := newTestEngine(t, &captured)

	var handlerLen int
	e.POST("/upload", func(c *gin.Context) {
		buf := &bytes.Buffer{}
		_, _ = buf.ReadFrom(c.Request.Body)
		handlerLen = buf.Len()
		c.String(http.StatusOK, "ok")
	})

	bigBody := strings.Repeat("y", 300_000)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader(bigBody))
	req.ContentLength = int64(len(bigBody))
	e.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, len(bigBody), handlerLen, "handler 应能读取完整上传体")
	assert.Len(t, captured, 1)
	assert.Equal(t, "<body too large to log>", captured[0].RequestBody)
}
