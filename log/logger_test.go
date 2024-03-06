package log

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

var logger, _ = NewLogger(&ZeroLoggerConfig{
	MaxSize:          10,
	MaxAge:           10,
	MaxBackups:       10,
	Compress:         false,
	Filename:         "./test.log",
	LogLevel:         "trace",
	CallerPathPrefix: "/home/charles/codes/mico-assist/library",
})

func TestNewLogger(t *testing.T) {
	multi := logger.Multi()
	multi.Trace().Msg("trace")
	multi.Debug().Msg("debug")
	logger.Info("info")
	logger.Warn("warn")
	logger.Error("error")

	file := logger.File()
	file.Error().Msg("file error")
	multi.Error().Err(errors.New("some error")).Msg("multi error")
	logger.TimeRecord(time.Now(), "time record %s", "...")
	logger.Fatal("fatal")
}

func TestZeroLogger_WithPipe(t *testing.T) {
	fmt.Fprintln(logger.ErrorPipe(), "error")
	fmt.Fprintln(logger.WithPipe(), "info")
	time.Sleep(time.Second * 10)
}

func TestZeroLogger_Panic(t *testing.T) {
	logger.Warn("warn")
	logger.Panic("panic")
}

func TestZeroLogger_Fatal(t *testing.T) {
	logger.Fatal("fatal")
}

func TestZeroLogger_SetLevel(t *testing.T) {
	t.Log(logger.GetLevel())
	logger.Tracef("trace00")
	multi := logger.Multi()
	multi.Trace().Msg("trace")
	multi.Debug().Msg("debug")
	t.Log("set level")
	if err := logger.SetLevel("warn"); err != nil {
		t.Error(err)
		return
	}
	logger.Tracef("trace01")
	multi.Trace().Msg("trace1")
	multi.Debug().Msg("debug1")
	t.Log(logger.GetLevel())
}
