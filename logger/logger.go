package logger

import (
	"go.uber.org/zap"
	"sync"
)

var once sync.Once
var instance *zap.Logger

func CreateLogger(cfg zap.Config) {
	tmp, err := cfg.Build()
	if err != nil {
		panic("log init error:" + err.Error())
	}
	instance = tmp
}
func Logger() *zap.Logger {
	return instance
}
func Sugar() *zap.SugaredLogger {
	return instance.Sugar()
}
