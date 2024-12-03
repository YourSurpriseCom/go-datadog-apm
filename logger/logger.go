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
	internalLoggger *zap.SugaredLogger
}

func NewLogger() Logger {
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

	zapLogger, _ := config.Build()

	return Logger{
		internalLoggger: zapLogger.Sugar(),
	}
}

func (log Logger) Debug(ctx context.Context, template string, args ...interface{}) {
	span, ok := tracer.SpanFromContext(ctx)
	if ok {
		log.internalLoggger.Debugw(fmt.Sprintf(template, args...), "dd.trace_id", span.Context().TraceID(), "dd.span_id", span.Context().SpanID())
	} else {
		log.internalLoggger.Debugf(template, args...)
	}
}

func (log Logger) Info(ctx context.Context, template string, args ...interface{}) {
	span, ok := tracer.SpanFromContext(ctx)
	if ok {
		log.internalLoggger.Infow(fmt.Sprintf(template, args...), "dd.trace_id", span.Context().TraceID(), "dd.span_id", span.Context().SpanID())
	} else {
		log.internalLoggger.Infof(template, args...)
	}
}

func (log Logger) Warn(ctx context.Context, template string, args ...interface{}) {
	span, ok := tracer.SpanFromContext(ctx)
	if ok {
		log.internalLoggger.Warnw(fmt.Sprintf(template, args...), "dd.trace_id", span.Context().TraceID(), "dd.span_id", span.Context().SpanID())
	} else {
		log.internalLoggger.Warnf(template, args...)
	}
}

func (log Logger) Error(ctx context.Context, template string, args ...interface{}) {
	span, ok := tracer.SpanFromContext(ctx)
	if ok {
		span.SetTag("error", fmt.Errorf(template, args...))
		log.internalLoggger.Errorw(fmt.Sprintf(template, args...), "dd.trace_id", span.Context().TraceID(), "dd.span_id", span.Context().SpanID())
	} else {
		log.internalLoggger.Errorf(template, args...)
	}
}

func (log Logger) Fatal(args ...interface{}) {
	log.internalLoggger.Fatal(args...)
}

func (log Logger) Sync() {
	err := log.internalLoggger.Sync()
	if err != nil {
		log.Fatal("unable to sync logs from buffer")
	}
}
