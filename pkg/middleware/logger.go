package middleware

import (
	"bytes"
	"context"
	"github.com/gin-gonic/gin"
	"aurora-admin/pkg/log"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type LogFormatterParams struct {
	gin.LogFormatterParams
	Ctx          context.Context
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
	// Optional.
	Skip gin.Skipper
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

	requestBody, err := io.ReadAll(params.Request.Body)
	if err != nil {
		logger.Error("read request body failed", "err", err)
	}

	logger.LogC(params.Ctx, level,
		params.Path,
		slog.Int("code", params.StatusCode),
		slog.String("method", params.Method),
		slog.Duration("latency", params.Latency),
		slog.String("client", params.ClientIP),
		slog.String("request", string(requestBody)),
		slog.String("response", params.ResponseBody.String()),
	)
}

var defaultSkipper = func(c *gin.Context) bool {
	return c.Request.Method == http.MethodGet
}

type responseWriter struct {
	gin.ResponseWriter
	buffer *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.buffer.Write(b)
	return w.ResponseWriter.Write(b)
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
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		buffer := bytes.NewBuffer(nil)
		c.Writer = &responseWriter{
			ResponseWriter: c.Writer,
			buffer:         buffer,
		}

		// Process request
		c.Next()

		// Log only when it is not being skipped
		if _, ok := skip[path]; ok || skipper(c) {
			return
		}

		param := LogFormatterParams{
			LogFormatterParams: gin.LogFormatterParams{
				Request: c.Request,
				Keys:    c.Keys,
			},
			Ctx:          c,
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

		if raw != "" {
			path = path + "?" + raw
		}

		param.Path = path

		formatter(param, logger)
	}
}
