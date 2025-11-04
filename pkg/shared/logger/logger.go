package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates a zap logger configured via env:
// APP_LOG_LEVEL: debug|info|warn|error (default: info)
// APP_LOG_ENCODING: json|console (default: json)
// APP_LOG_SAMPLING: true|false (default: true)
func New() (*zap.Logger, error) {
	levelStr := strings.ToLower(strings.TrimSpace(os.Getenv("APP_LOG_LEVEL")))
	enc := strings.ToLower(strings.TrimSpace(os.Getenv("APP_LOG_ENCODING")))
	samplingStr := strings.ToLower(strings.TrimSpace(os.Getenv("APP_LOG_SAMPLING")))

	lvl := zapcore.InfoLevel
	switch levelStr {
	case "debug":
		lvl = zapcore.DebugLevel
	case "warn":
		lvl = zapcore.WarnLevel
	case "error":
		lvl = zapcore.ErrorLevel
	default:
		lvl = zapcore.InfoLevel
	}
	if enc == "" {
		enc = "json"
	}

	cfg := zap.Config{
		Level:    zap.NewAtomicLevelAt(lvl),
		Encoding: enc,
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stack",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.EpochTimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	// Optional sampling (reduce log volume under load)
	if samplingStr == "false" {
		cfg.Sampling = nil
	} else {
		cfg.Sampling = &zap.SamplingConfig{Initial: 100, Thereafter: 100}
	}
	return cfg.Build()
}
