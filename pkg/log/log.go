package log

import (
	"context"
	"fmt"
	"aurora-admin/pkg/util/errutil"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const slogFields = "slog_fields"

var logger = NewLogger("")

func DefaultLogger() *Logger {
	return logger
}

func SetLevel(level string) error {
	return logger.SetLevel(level)
}

func With(args ...any) *Logger {
	return logger.With(args...)
}

func Debug(msg string, args ...any) {
	logger.log(context.Background(), slog.LevelDebug, msg, args...)
}

func DebugC(ctx context.Context, msg string, args ...any) {
	logger.log(ctx, slog.LevelDebug, msg, args...)
}

func Info(msg string, args ...any) {
	logger.log(context.Background(), slog.LevelInfo, msg, args...)
}

func InfoC(ctx context.Context, msg string, args ...any) {
	logger.log(ctx, slog.LevelInfo, msg, args...)
}

func Warn(msg string, args ...any) {
	logger.log(context.Background(), slog.LevelWarn, msg, args...)
}

func WarnC(ctx context.Context, msg string, args ...any) {
	logger.log(ctx, slog.LevelWarn, msg, args...)
}

func Error(msg string, args ...any) {
	logger.log(context.Background(), slog.LevelError, msg, args...)
}

func ErrorC(ctx context.Context, msg string, args ...any) {
	logger.log(ctx, slog.LevelError, msg, args...)
}

func DebugF(format string, a ...any) {
	logger.logF(context.Background(), slog.LevelDebug, format, a...)
}

func DebugFC(ctx context.Context, format string, a ...any) {
	logger.logF(ctx, slog.LevelDebug, format, a...)
}

func InfoF(format string, a ...any) {
	logger.logF(context.Background(), slog.LevelInfo, format, a...)
}

func InfoFC(ctx context.Context, format string, a ...any) {
	logger.logF(ctx, slog.LevelInfo, format, a...)
}

func WarnF(format string, a ...any) {
	logger.logF(context.Background(), slog.LevelWarn, format, a...)
}

func WarnFC(ctx context.Context, format string, a ...any) {
	logger.logF(ctx, slog.LevelWarn, format, a...)
}

func ErrorF(format string, a ...any) {
	logger.logF(context.Background(), slog.LevelError, format, a...)
}

func ErrorFC(ctx context.Context, format string, a ...any) {
	logger.logF(ctx, slog.LevelError, format, a...)
}

func Close(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		logger.Log(slog.LevelError, "close failed", "err", err)
	}
}

type Logger struct {
	logger *slog.Logger
	level  *slog.LevelVar
}

func (l *Logger) SetLevel(level string) error {
	err := l.level.UnmarshalText([]byte(level))
	if err != nil {
		return errutil.Wrap(err)
	}
	return nil
}

func (l *Logger) With(args ...any) *Logger {
	return &Logger{logger: l.logger.With(args...), level: l.level}
}

func (l *Logger) Log(level slog.Level, msg string, args ...any) {
	l.log(context.Background(), level, msg, args...)
}

func (l *Logger) LogC(ctx context.Context, level slog.Level, msg string, args ...any) {
	l.log(ctx, level, msg, args...)
}

func (l *Logger) Debug(msg string, args ...any) {
	l.log(context.Background(), slog.LevelDebug, msg, args...)
}

func (l *Logger) DebugC(ctx context.Context, msg string, args ...any) {
	l.log(ctx, slog.LevelDebug, msg, args...)
}

func (l *Logger) Info(msg string, args ...any) {
	l.log(context.Background(), slog.LevelInfo, msg, args...)
}

func (l *Logger) InfoC(ctx context.Context, msg string, args ...any) {
	l.log(ctx, slog.LevelInfo, msg, args...)
}

func (l *Logger) Warn(msg string, args ...any) {
	l.log(context.Background(), slog.LevelWarn, msg, args...)
}

func (l *Logger) WarnC(ctx context.Context, msg string, args ...any) {
	l.log(ctx, slog.LevelWarn, msg, args...)
}

func (l *Logger) Error(msg string, args ...any) {
	l.log(context.Background(), slog.LevelError, msg, args...)
}

func (l *Logger) ErrorC(ctx context.Context, msg string, args ...any) {
	l.log(ctx, slog.LevelError, msg, args...)
}

func (l *Logger) LogF(level slog.Level, format string, a ...any) {
	l.logF(context.Background(), level, format, a...)
}

func (l *Logger) LogFC(ctx context.Context, level slog.Level, format string, a ...any) {
	l.logF(ctx, level, format, a...)
}

func (l *Logger) DebugF(format string, a ...any) {
	l.logF(context.Background(), slog.LevelDebug, format, a...)
}

func (l *Logger) DebugFC(ctx context.Context, format string, a ...any) {
	l.logF(ctx, slog.LevelDebug, format, a...)
}

func (l *Logger) InfoF(format string, a ...any) {
	l.logF(context.Background(), slog.LevelInfo, format, a...)
}

func (l *Logger) InfoFC(ctx context.Context, format string, a ...any) {
	l.logF(ctx, slog.LevelInfo, format, a...)
}

func (l *Logger) WarnF(format string, a ...any) {
	l.logF(context.Background(), slog.LevelWarn, format, a...)
}

func (l *Logger) WarnFC(ctx context.Context, format string, a ...any) {
	l.logF(ctx, slog.LevelWarn, format, a...)
}

func (l *Logger) ErrorF(format string, a ...any) {
	l.logF(context.Background(), slog.LevelError, format, a...)
}

func (l *Logger) ErrorFC(ctx context.Context, format string, a ...any) {
	l.logF(ctx, slog.LevelError, format, a...)
}

func (l *Logger) log(ctx context.Context, level slog.Level, msg string, args ...any) {
	if !l.logger.Enabled(ctx, level) {
		return
	}
	var pcs [1]uintptr
	// skip [runtime.Callers, this function, this function's caller]
	runtime.Callers(3, pcs[:])
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.Add(args...)
	if ctx == nil {
		ctx = context.Background()
	}
	_ = l.logger.Handler().Handle(ctx, r)
}

func (l *Logger) logF(ctx context.Context, level slog.Level, format string, a ...any) {
	if !l.logger.Enabled(ctx, level) {
		return
	}
	var pcs [1]uintptr
	// skip [runtime.Callers, this function, this function's caller]
	runtime.Callers(3, pcs[:])
	r := slog.NewRecord(time.Now(), level, fmt.Sprintf(format, a...), pcs[0])
	if ctx == nil {
		ctx = context.Background()
	}
	_ = l.logger.Handler().Handle(ctx, r)
}

func NewLogger(path string) *Logger {
	var w io.Writer = os.Stdout
	if path != "" {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o640)
		errutil.Fatal(err)
		w = io.MultiWriter(w, file)
	}

	lvl := &slog.LevelVar{}

	h := &ContextHandler{
		slog.NewTextHandler(w, &slog.HandlerOptions{
			AddSource: true,
			Level:     lvl,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					source := a.Value.Any().(*slog.Source)
					//source.File = source.Function[strings.LastIndex(source.Function, "/")+1:]
					dir, file := filepath.Split(source.File)
					source.File = fmt.Sprintf("%s/%s", filepath.Base(dir), file)
				}
				return a
			},
		}),
	}
	lg := slog.New(h)
	return &Logger{logger: lg, level: lvl}
}

type ContextHandler struct {
	slog.Handler
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	args, ok := ctx.Value(slogFields).([]any)
	if ok {
		r.Add(args...)
	}
	return h.Handler.Handle(ctx, r)
}

func AppendCtx(ctx context.Context, args ...any) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	curArgs, ok := ctx.Value(slogFields).([]any)
	if ok {
		args = append(curArgs, args...)
	}
	return context.WithValue(ctx, slogFields, args)
}

func Init(path string, level string) {
	logger = NewLogger(path)
	err := logger.SetLevel(level)
	errutil.Fatal(err)
	logger.Warn("init log", "level", level, "path", path)
}
