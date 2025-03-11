package logger

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type Logger struct {
	internalLogger *zap.SugaredLogger
}

type LoggerOption func(*Logger)

func WithConfig(config zap.Config) LoggerOption {
	return func(l *Logger) {
		zapLogger, err := config.Build()
		if err != nil {
			fmt.Println("Error building logger", err.Error())
			panic(err)
		}
		l.internalLogger = zapLogger.Sugar()
	}
}

func NewLogger(options ...LoggerOption) Logger {
	logger := Logger{}

	for _, option := range options {
		option(&logger)
	}

	if logger.internalLogger == nil {
		logger.internalLogger = defaultLogger()
	}

	return logger
}

func defaultLogger() *zap.SugaredLogger {
	var logLevel zapcore.Level
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		logLevel = zapcore.DebugLevel
	case "info":
		logLevel = zapcore.InfoLevel
	case "warning":
		logLevel = zapcore.WarnLevel
	case "fatal":
		logLevel = zapcore.FatalLevel
	default:
		logLevel = zapcore.InfoLevel
	}

	config := zap.Config{
		Level:       zap.NewAtomicLevelAt(logLevel),
		Development: false,
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, _ := config.Build()
	return logger.Sugar()
}

func (log Logger) Debug(ctx context.Context, template string, args ...interface{}) {
	span, ok := tracer.SpanFromContext(ctx)
	if ok {
		log.internalLogger.Debugw(fmt.Sprintf(template, args...), "dd.trace_id", span.Context().TraceID(), "dd.span_id", span.Context().SpanID())
	} else {
		log.internalLogger.Debugf(template, args...)
	}
}

func (log Logger) Info(ctx context.Context, template string, args ...interface{}) {
	span, ok := tracer.SpanFromContext(ctx)
	if ok {
		log.internalLogger.Infow(fmt.Sprintf(template, args...), "dd.trace_id", span.Context().TraceID(), "dd.span_id", span.Context().SpanID())
	} else {
		log.internalLogger.Infof(template, args...)
	}
}

func (log Logger) Warn(ctx context.Context, template string, args ...interface{}) {
	span, ok := tracer.SpanFromContext(ctx)
	if ok {
		log.internalLogger.Warnw(fmt.Sprintf(template, args...), "dd.trace_id", span.Context().TraceID(), "dd.span_id", span.Context().SpanID())
	} else {
		log.internalLogger.Warnf(template, args...)
	}
}

func (log Logger) Error(ctx context.Context, template string, args ...interface{}) {
	span, ok := tracer.SpanFromContext(ctx)
	if ok {
		span.SetTag("error", fmt.Errorf(template, args...))
		log.internalLogger.Errorw(fmt.Sprintf(template, args...), "dd.trace_id", span.Context().TraceID(), "dd.span_id", span.Context().SpanID())
	} else {
		log.internalLogger.Errorf(template, args...)
	}
}

func (log Logger) Fatal(args ...interface{}) {
	log.internalLogger.Fatal(args...)
}

func (log Logger) Sync() {
	err := log.internalLogger.Sync()
	if err != nil {
		log.Fatal("unable to sync logs from buffer")
	}
}
