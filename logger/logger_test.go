package logger

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"testing"
)

var configStr = []byte(` {
		"level": "debug",
		"encoding": "json",
		"outputPaths": ["stdout", "../log/log_test" ],
	    "errOutputPaths": ["stderr", "../log/err_test"],
	    "initialFields": {"foo": "bar"},
	    "encoderConfig": {
	        "messageKey": "m",
	        "levelKey": "l",
	        "levelEncoder": "lowercase"
	    }
	}`)

func TestCreateLogger(t *testing.T) {
	cfg := zap.Config{}
	err := json.Unmarshal(configStr, &cfg)
	assert.Nil(t, err)
	type args struct {
		cfg zap.Config
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "success test", args: args{cfg: cfg}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}

func TestLogger(t *testing.T) {
	cfg := zap.Config{}
	err := json.Unmarshal(configStr, &cfg)
	assert.Nil(t, err)

	CreateLogger(cfg)
	assert.NotNil(t, instance)
	assert.Equal(t, Logger(), instance)
}

func TestSugar(t *testing.T) {
	cfg := zap.Config{}
	err := json.Unmarshal(configStr, &cfg)
	assert.Nil(t, err)

	CreateLogger(cfg)
	assert.NotNil(t, instance)
	assert.IsType(t, &zap.SugaredLogger{}, Sugar())
}
