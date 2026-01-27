package logger

import (
	"net/http"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func Init(development bool) error {
	var err error
	if development {
		config := zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006/01/02 15:04:05")
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		Logger, err = config.Build()

	} else {
		config := zap.NewProductionConfig()
		config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006/01/02 15:04:05")
		Logger, err = config.Build()
	}

	if err != nil {
		panic(err)
	}

	return nil
}

func Sync() {
	Logger.Sync()
}
func Info(msg string, fields ...zap.Field) {
	Logger.Info(msg, fields...)
}

func Log(lvl zapcore.Level, msg string, fields ...zap.Field){
	Logger.Log(lvl, msg, fields...)
}

func HttpRequestInfo(r *http.Request, msg string, fields ...zap.Field) {
	
	allFields := []zap.Field{
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.String("query", r.URL.RawQuery),
		zap.String("client_ip", r.RemoteAddr),
	}
	allFields = append(allFields, fields...)
	Logger.Info(msg, allFields...)
}

func Error(msg string, err error, fields ...zap.Field) {
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	Logger.Error(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	Logger.Warn(msg, fields...)
}
