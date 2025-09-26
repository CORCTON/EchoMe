package config

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var ( // 全局日志实例
	Logger *zap.Logger
)

// InitLogger 初始化全局zap日志
func InitLogger() {
	// 配置zap日志
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "time"
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.NameKey = "logger"
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.MessageKey = "msg"
	config.EncoderConfig.StacktraceKey = "stacktrace"
	config.EncoderConfig.LineEnding = zapcore.DefaultLineEnding
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	// 构建日志器
	logger, err := config.Build()
	if err != nil {
		panic("初始化日志失败: " + err.Error())
	}

	// 设置全局日志实例
	Logger = logger

	// 替换标准库的log包
	zap.ReplaceGlobals(logger)
}
