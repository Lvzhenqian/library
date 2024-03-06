package log

import (
	"bufio"
	"context"
	"fmt"
	"github.com/Lvzhenqian/library/errors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type colors int

const (
	Error colors = 31 + iota
	Info
	Panic
	_
	Fatal
	Debug
	Trace
	_
	Weak colors = 2
	Bold colors = 1
	Warn        = Panic
)

var Dict = zerolog.Dict

type ZeroLoggerConfig struct {
	// 每个日志文件最大多少字节
	MaxSize int
	// 最大保存多少天前的日志
	MaxAge int
	// 最大保留多少个旧日志
	MaxBackups int
	// 是否压缩旧日志
	Compress bool
	// Filename 文件路径名
	Filename string
	// LogLevel 日志级别
	LogLevel string
	// CallerPathPrefix stdout输出文件名路径忽略前缀。
	// 可以通过环境变量名：CONSOLE_CALLER_PATH_PREFIX 修改
	CallerPathPrefix string
}

type ZeroLogger struct {
	file        zerolog.Context
	multi       zerolog.Context
	fileWriter  io.Writer
	multiWriter io.Writer
	level       zerolog.Level
}

func consoleFormatCaller(prefix string) zerolog.Formatter {
	return func(i interface{}) string {
		var c string
		if cc, ok := i.(string); ok {
			c = cc
		}
		if len(c) > 0 {
			if rel, err := filepath.Rel(prefix, c); err == nil {
				c = rel
			}
			c += fmt.Sprintf("\x1b[%d;%dm%s\x1b[0m", Debug, Weak, " >")
		}
		return c
	}
}

func NewLogger(conf *ZeroLoggerConfig) (*ZeroLogger, error) {

	fileWriter := &lumberjack.Logger{
		Filename:   conf.Filename,
		MaxSize:    conf.MaxSize,
		MaxAge:     conf.MaxAge,
		MaxBackups: conf.MaxBackups,
		LocalTime:  true,
		Compress:   conf.Compress,
	}

	level, err := zerolog.ParseLevel(conf.LogLevel)
	if err != nil {
		return nil, err
	}
	zerolog.ErrorStackMarshaler = func(err error) interface{} {
		return errors.ErrorStack(err)
	}

	consoleCallerPrefix := conf.CallerPathPrefix
	if prefix, ok := os.LookupEnv("CONSOLE_CALLER_PATH_PREFIX"); ok {
		consoleCallerPrefix = prefix
	}

	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
		FormatLevel: func(i interface{}) string {
			value, ok := i.(string)
			if !ok {
				return fmt.Sprintf("%4s", i)
			}
			return colorLevel(value)
		},
		FormatErrFieldName: func(i interface{}) string {
			value, ok := i.(string)
			if !ok {
				return fmt.Sprintf("%4s", i)
			}
			return fmt.Sprintf("\x1b[%d;%dm%s\x1b[0m=", Warn, Weak, value)
		},
		FormatCaller: consoleFormatCaller(consoleCallerPrefix),
	}

	multiWriter := zerolog.MultiLevelWriter(consoleWriter, fileWriter)
	file := zerolog.New(fileWriter).Level(level).With().Timestamp().CallerWithSkipFrameCount(zerolog.CallerSkipFrameCount)
	multi := zerolog.New(multiWriter).Level(level).With().Timestamp().CallerWithSkipFrameCount(zerolog.CallerSkipFrameCount)
	return &ZeroLogger{
		file:  file,
		multi: multi,
		level: level,
	}, nil
}

func colorLevel(s string) string {

	format := func(color, style colors) string {

		return fmt.Sprintf("|\x1b[%d;%dm%-5s\x1b[0m|", color, style, strings.Title(s))
	}
	same := func(v string) bool {
		return strings.EqualFold(s, v)
	}
	switch {
	case same("panic"):
		return format(Panic, Bold)
	case same("fatal"):
		return format(Fatal, Bold)
	case same("error"):
		return format(Error, Bold)
	case same("warn"):
		return format(Warn, Weak)
	case same("info"):
		return format(Info, Bold)
	case same("debug"):
		return format(Debug, Bold)
	default:
		return format(Trace, Bold)
	}
}

func (l *ZeroLogger) SetLevel(level string) error {
	newLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		return err
	}
	l.file = zerolog.New(l.fileWriter).Level(newLevel).With().Timestamp()
	l.multi = zerolog.New(l.multiWriter).Level(newLevel).With().Timestamp()
	l.level = newLevel
	return nil
}
func (l *ZeroLogger) GetLevel() string {
	return l.level.String()
}
func (l *ZeroLogger) File() zerolog.Logger {
	return l.file.Logger()
}
func (l *ZeroLogger) FileWithSkipFrame(i int) zerolog.Logger {
	return l.file.CallerWithSkipFrameCount(i).Logger()
}
func (l *ZeroLogger) Multi() zerolog.Logger {
	return l.multi.Logger()
}
func (l *ZeroLogger) MultiWithSkipFrame(i int) zerolog.Logger {
	return l.multi.CallerWithSkipFrameCount(i).Logger()
}
func (l *ZeroLogger) Panic(msg string) {
	multi := l.MultiWithSkipFrame(3)
	multi.Panic().Stack().Msg(msg)
}
func (l *ZeroLogger) Panicf(f string, value ...interface{}) {
	multi := l.MultiWithSkipFrame(3)
	multi.Panic().Stack().Msgf(f, value...)
}
func (l *ZeroLogger) Fatal(msg string) {
	multi := l.MultiWithSkipFrame(3)
	multi.Fatal().Stack().Msg(msg)
}
func (l *ZeroLogger) Fatalf(f string, value ...interface{}) {
	multi := l.MultiWithSkipFrame(3)
	multi.Fatal().Stack().Msgf(f, value...)
}
func (l *ZeroLogger) Error(msg string) {
	multi := l.MultiWithSkipFrame(3)
	multi.Error().Stack().Msg(msg)
}
func (l *ZeroLogger) Errorf(f string, value ...interface{}) {
	multi := l.MultiWithSkipFrame(3)
	multi.Error().Stack().Msgf(f, value...)
}
func (l *ZeroLogger) WithError(err error, msg string) {
	multi := l.MultiWithSkipFrame(3)
	multi.Error().Stack().Err(err).Msg(msg)
}
func (l *ZeroLogger) WithErrorf(err error, format string, args ...interface{}) {
	multi := l.MultiWithSkipFrame(3)
	multi.Error().Stack().Err(err).Msgf(format, args...)
}
func (l *ZeroLogger) Warn(msg string) {
	multi := l.MultiWithSkipFrame(3)
	multi.Warn().Msg(msg)
}
func (l *ZeroLogger) Warnf(f string, value ...interface{}) {
	multi := l.MultiWithSkipFrame(3)
	multi.Warn().Msgf(f, value...)
}
func (l *ZeroLogger) Info(msg string) {
	file := l.FileWithSkipFrame(3)
	file.Info().Msg(msg)
}
func (l *ZeroLogger) Infof(f string, value ...interface{}) {
	file := l.FileWithSkipFrame(3)
	file.Info().Msgf(f, value...)
}
func (l *ZeroLogger) Debug(msg string) {
	file := l.FileWithSkipFrame(3)
	file.Debug().Msg(msg)
}
func (l *ZeroLogger) Debugf(f string, value ...interface{}) {
	file := l.FileWithSkipFrame(3)
	file.Debug().Msgf(f, value...)
}
func (l *ZeroLogger) Trace(msg string) {
	file := l.FileWithSkipFrame(3)
	file.Trace().Msg(msg)
}
func (l *ZeroLogger) Tracef(f string, value ...interface{}) {
	file := l.FileWithSkipFrame(3)
	file.Trace().Msgf(f, value...)
}

func (l *ZeroLogger) WithWarp(err error, msg string) error {
	e := errors.Wrap(err)
	multi := l.MultiWithSkipFrame(3)
	multi.Error().Err(e).Msg(msg)
	return e
}

func (l *ZeroLogger) WithWarpf(err error, format string, args ...interface{}) error {
	e := errors.Wrapf(err, format, args...)
	multi := l.MultiWithSkipFrame(3)
	multi.Error().Err(e).Msgf(format, args...)
	return e
}

// WithPipe 会导致 1个goroutine 泄漏，请尽量少用
func (l *ZeroLogger) WithPipe() *io.PipeWriter {
	r, w := io.Pipe()
	go func() {
		scan := bufio.NewScanner(r)
		for scan.Scan() {
			l.Infof("pipe writer: %s", scan.Text())
		}
	}()
	return w
}

// ErrorPipe 会导致 1个goroutine 泄漏，请尽量少用
func (l *ZeroLogger) ErrorPipe() *io.PipeWriter {
	r, w := io.Pipe()
	go func() {
		defer func() {
			fmt.Println("thread exit..")
		}()
		scan := bufio.NewScanner(r)
		for scan.Scan() {
			l.Errorf("pipe writer: %s", scan.Text())
		}
	}()
	return w
}

func (l *ZeroLogger) TimeRecord(t time.Time, f string, value ...interface{}) {
	multi := l.MultiWithSkipFrame(3)
	multi.Info().Str("Since", time.Since(t).String()).Msgf(f, value...)
}

func (l *ZeroLogger) WithCtx(ctx context.Context) *ZeroLogger {
	s := trace.SpanContextFromContext(ctx)
	if s.HasTraceID() {
		traceId := s.TraceID().String()
		spanId := s.SpanID().String()
		hook := traceHook{
			trace_id: traceId,
			span_id:  spanId,
		}
		l.multi = l.multi.Logger().Hook(hook).With()
		l.file = l.file.Logger().Hook(hook).With()
	}
	return l
}

type traceHook struct {
	trace_id string
	span_id  string
}

func (h traceHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	if level != zerolog.NoLevel {
		e.Str("trace_id", h.trace_id).Str("span_id", h.span_id)
	}
}
