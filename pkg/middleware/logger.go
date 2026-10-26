package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"aurora-admin/pkg/log"
)

// maxLoggedBodySize 限制记录到日志的请求/响应体大小。
// 存档上传/下载等大文件接口的 body 不应被整体缓冲进内存。
const maxLoggedBodySize = 64 << 10 // 64KB

type LogFormatterParams struct {
	gin.LogFormatterParams
	Ctx          context.Context
	RequestBody  string
	ResponseBody *bytes.Buffer
}

type LogFormatter func(params LogFormatterParams, logger *log.Logger)

type LoggerConfig struct {
	// Optional. Default value is gin.defaultLogFormatter
	Formatter LogFormatter

	// SkipPaths is an url path array which logs are not written.
	// Optional.
	SkipPaths []string

	// Skip is a Skipper that indicates which logs should not be written.
	// 在请求处理前调用，只能基于请求本身（method/path 等）判断。
	// Optional.
	Skip gin.Skipper
}

// sensitiveKeyPattern 匹配需要脱敏的 JSON 字段名（密码、令牌、密钥等），
// 避免登录密码、JWT、集群密码等以明文形式落进日志文件。
var sensitiveKeyPattern = regexp.MustCompile(`(?i)pass|token|secret|jwt|authorization|cookie`)

// maskSensitiveJSON 对 JSON 请求/响应体做字段级脱敏。
// 非 JSON 内容（表单、二进制等）原样返回，避免影响日志可读性。
func maskSensitiveJSON(body string) string {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return body
	}
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		return body
	}

	var v any
	if err := json.Unmarshal([]byte(trimmed), &v); err != nil {
		// 可能因日志截断导致 JSON 不完整，这种情况下无法安全脱敏，直接丢弃内容
		return "<unparsable body, omitted>"
	}

	masked, err := json.Marshal(maskValue(v))
	if err != nil {
		return "<unparsable body, omitted>"
	}
	return string(masked)
}

func maskValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if sensitiveKeyPattern.MatchString(k) {
				t[k] = "***"
				continue
			}
			t[k] = maskValue(val)
		}
		return t
	case []any:
		for i := range t {
			t[i] = maskValue(t[i])
		}
		return t
	default:
		return v
	}
}

var defaultLogFormatter = func(params LogFormatterParams, logger *log.Logger) {
	var level slog.Level
	if params.StatusCode < http.StatusBadRequest {
		level = slog.LevelInfo
	} else {
		level = slog.LevelError
	}

	if params.Latency > time.Minute {
		params.Latency = params.Latency.Truncate(time.Second)
	}

	logger.LogC(params.Ctx, level,
		params.Path,
		slog.Int("code", params.StatusCode),
		slog.String("method", params.Method),
		slog.Duration("latency", params.Latency),
		slog.String("client", params.ClientIP),
		slog.String("request", maskSensitiveJSON(params.RequestBody)),
		slog.String("response", maskSensitiveJSON(params.ResponseBody.String())),
	)
}

var defaultSkipper = func(c *gin.Context) bool {
	return c.Request.Method == http.MethodGet
}

// responseWriter 透传响应的同时，仅保留前 maxLoggedBodySize 字节用于日志。
type responseWriter struct {
	gin.ResponseWriter
	buffer *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if remain := maxLoggedBodySize - w.buffer.Len(); remain > 0 {
		if len(b) < remain {
			remain = len(b)
		}
		w.buffer.Write(b[:remain])
	}
	return w.ResponseWriter.Write(b)
}

// snapshotRequestBody 读取（有上限的）请求体用于日志，并将原 body 塞回供后续 handler 使用。
func snapshotRequestBody(req *http.Request) string {
	if req.Body == nil || req.Body == http.NoBody {
		return ""
	}
	if req.ContentLength > maxLoggedBodySize {
		return "<body too large to log>"
	}

	b, err := io.ReadAll(io.LimitReader(req.Body, maxLoggedBodySize+1))
	if err != nil {
		return "<read body failed>"
	}
	if len(b) > maxLoggedBodySize {
		return "<body too large to log>"
	}
	req.Body = io.NopCloser(bytes.NewReader(b))
	return string(b)
}

func Logger(conf LoggerConfig, logger *log.Logger) gin.HandlerFunc {
	formatter := conf.Formatter
	if formatter == nil {
		formatter = defaultLogFormatter
	}
	skipper := conf.Skip
	if skipper == nil {
		skipper = defaultSkipper
	}

	notlogged := conf.SkipPaths
	var skip map[string]struct{}
	if length := len(notlogged); length > 0 {
		skip = make(map[string]struct{}, length)
		for _, path := range notlogged {
			skip[path] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		// 在处理请求前判断是否跳过：跳过的请求不挂响应缓冲、不读请求体，零额外开销。
		// 注意：与原实现一致，SkipPaths 匹配不含 query 的 URL.Path。
		if _, ok := skip[c.Request.URL.Path]; ok || skipper(c) {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		if raw != "" {
			path = path + "?" + raw
		}

		requestBody := maskSensitiveJSON(snapshotRequestBody(c.Request))

		// Start timer
		start := time.Now()

		buffer := bytes.NewBuffer(nil)
		c.Writer = &responseWriter{
			ResponseWriter: c.Writer,
			buffer:         buffer,
		}

		// Process request
		c.Next()

		param := LogFormatterParams{
			LogFormatterParams: gin.LogFormatterParams{
				Request: c.Request,
				Keys:    c.Keys,
			},
			Ctx:          c,
			RequestBody:  requestBody,
			ResponseBody: buffer,
		}

		// Stop timer
		param.TimeStamp = time.Now()
		param.Latency = param.TimeStamp.Sub(start)

		param.ClientIP = c.ClientIP()
		param.Method = c.Request.Method
		param.StatusCode = c.Writer.Status()
		param.ErrorMessage = c.Errors.ByType(gin.ErrorTypePrivate).String()

		param.BodySize = c.Writer.Size()
		param.Path = path

		formatter(param, logger)
	}
}
