package logger

import (
	"encoding/json"
	"go.uber.org/zap"
)

type LoggerFake struct {
	*zap.Logger
}

var configFake = []byte(` {
		"level": "debug",
		"encoding": "json",
		"outputPaths": ["stdout", "../log/log_fake" ],
	    "errOutputPaths": ["stderr", "../log/err_fake"],
	    "initialFields": {"foo": "bar"},
	    "encoderConfig": {
	        "messageKey": "m",
	        "levelKey": "l",
	        "levelEncoder": "lowercase"
	    }
	}`)

func CreateFakeLogger() {
	cfg := zap.Config{}
	_ = json.Unmarshal(configFake, &cfg)
	CreateLogger(cfg)
}
