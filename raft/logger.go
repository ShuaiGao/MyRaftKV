package raft

import (
	"go.uber.org/zap"
	"sync"
)

type Logger interface {
	Debug(v ...interface{})
	Debugf(format string, v ...interface{})

	Info(v ...interface{})
	Infof(format string, v ...interface{})

	Warn(v ...interface{})
	Warnf(format string, v ...interface{})

	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})

	Panic(v ...interface{})
	Panicf(format string, v ...interface{})
}

type DefaultLogger struct {
	*zap.SugaredLogger
	debug bool
}

var (
	defaultLogger = &DefaultLogger{}
	raftLoggerMu  sync.Mutex
	raftLogger    = Logger(defaultLogger)
)

func getLogger() Logger {
	raftLoggerMu.Lock()
	defer raftLoggerMu.Unlock()
	return raftLogger
}
