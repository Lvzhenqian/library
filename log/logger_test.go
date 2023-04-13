package log

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

var logger, _ = NewLogger(&ZeroLoggerConfig{
	MaxSize:              10,
	MaxAge:               10,
	MaxBackups:           10,
	CallerSkipFrameCount: 3,
	Compress:             false,
	Filename:             "./test.log",
	LogLevel:             "trace",
	CallerPathPrefix:     "/home/charles/codes/mico-assist/library",
})

func TestNewLogger(t *testing.T) {
	logger.Multi().Trace().Msg("trace")
	logger.Multi().Debug().Msg("debug")
	logger.Info("info")
	logger.Warn("warn")
	logger.Error("error")

	logger.File().Error().Msg("file error")
	logger.Multi().Error().Err(errors.New("some error")).Msg("multi error")
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
	logger.Multi().Trace().Msg("trace")
	logger.Multi().Debug().Msg("debug")
	if err := logger.SetLevel("warn"); err != nil {
		t.Error(err)
		return
	}
	logger.Tracef("trace01")
	logger.Multi().Trace().Msg("trace1")
	logger.Multi().Debug().Msg("debug1")
	t.Log(logger.GetLevel())
}
