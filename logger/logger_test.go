package logger

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/mocktracer"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func setupLogsCapture() (*zap.SugaredLogger, *observer.ObservedLogs) {
	core, logs := observer.New(zap.InfoLevel)
	return zap.New(core).Sugar().WithOptions(zap.WithFatalHook(zapcore.WriteThenPanic)), logs
}

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name          string
		logLevel      string
		expectedLevel zapcore.Level
	}{
		{
			name:          "default level when no env var",
			logLevel:      "",
			expectedLevel: zapcore.InfoLevel,
		},
		{
			name:          "debug level",
			logLevel:      "debug",
			expectedLevel: zapcore.DebugLevel,
		},
		{
			name:          "info level",
			logLevel:      "info",
			expectedLevel: zapcore.InfoLevel,
		},
		{
			name:          "warning level",
			logLevel:      "warning",
			expectedLevel: zapcore.WarnLevel,
		},
		{
			name:          "fatal level",
			logLevel:      "fatal",
			expectedLevel: zapcore.FatalLevel,
		},
		{
			name:          "invalid level defaults to info",
			logLevel:      "invalid",
			expectedLevel: zapcore.InfoLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear env var before each test
			os.Unsetenv("LOG_LEVEL")
			if tt.logLevel != "" {
				os.Setenv("LOG_LEVEL", tt.logLevel)
			}

			logger := NewLogger()

			// Test log level
			if logger.internalLoggger.Desugar().Core().Enabled(tt.expectedLevel) != true {
				t.Errorf("Expected log level %v to be enabled", tt.expectedLevel)
			}

			// Test one level above should be disabled (except Fatal which is always highest)
			if tt.expectedLevel != zapcore.FatalLevel {
				nextLevel := tt.expectedLevel - 1
				if logger.internalLoggger.Desugar().Core().Enabled(nextLevel) != false {
					t.Errorf("Expected log level %v to be disabled", nextLevel)
				}
			}
		})
	}
}

func TestLogFunctions(t *testing.T) {
	mt := mocktracer.Start()
	defer mt.Stop()

	ctx := context.Background()
	span, spanContext := tracer.StartSpanFromContext(ctx, "test.span")
	span.Finish()

	tests := []struct {
		name            string
		ctx             context.Context
		msg             string
		logLevel        zapcore.Level
		wantTrace       bool
		expectedTraceId uint64
		expectedSpanId  uint64
	}{
		{
			name:            "debug with trace context",
			logLevel:        zap.DebugLevel,
			ctx:             spanContext,
			msg:             "debug test message: %s",
			wantTrace:       true,
			expectedTraceId: 124,
			expectedSpanId:  124,
		},
		{
			name:      "debug without trace context",
			logLevel:  zap.DebugLevel,
			ctx:       context.Background(),
			msg:       "debug test message: %s",
			wantTrace: false,
		},
		{
			name:            "info with trace context",
			logLevel:        zap.InfoLevel,
			ctx:             spanContext,
			msg:             "info test message: %s",
			wantTrace:       true,
			expectedTraceId: 124,
			expectedSpanId:  124,
		},
		{
			name:      "info without trace context",
			logLevel:  zap.InfoLevel,
			ctx:       context.Background(),
			msg:       "info test message: %s",
			wantTrace: false,
		},
		{
			name:            "warning with trace context",
			logLevel:        zap.WarnLevel,
			ctx:             spanContext,
			msg:             "warning test message: %s",
			wantTrace:       true,
			expectedTraceId: 124,
			expectedSpanId:  124,
		},
		{
			name:      "Warning without trace context",
			logLevel:  zap.WarnLevel,
			ctx:       context.Background(),
			msg:       "warning test message: %s",
			wantTrace: false,
		},
		{
			name:            "error with trace context",
			logLevel:        zap.ErrorLevel,
			ctx:             spanContext,
			msg:             "error test message: %s",
			wantTrace:       true,
			expectedTraceId: 124,
			expectedSpanId:  124,
		},
		{
			name:      "error without trace context",
			logLevel:  zap.ErrorLevel,
			ctx:       context.Background(),
			msg:       "error test message: %s",
			wantTrace: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			captureLog, logs := setupLogsCapture()
			logger := Logger{
				internalLoggger: captureLog,
			}

			switch tt.logLevel {
			case zap.DebugLevel:
				logger.Debug(tt.ctx, tt.msg, tt.name)
			case zap.InfoLevel:
				logger.Info(tt.ctx, tt.msg, tt.name)
			case zap.WarnLevel:
				logger.Warn(tt.ctx, tt.msg, tt.name)
			case zap.ErrorLevel:
				logger.Error(tt.ctx, tt.msg, tt.name)
			}

			logger.Sync()

			for _, logEntry := range logs.All() {
				if logEntry.Message != fmt.Sprintf(tt.msg, tt.name) {
					t.Errorf("Message incorect, expected '%s' got '%s'", tt.msg, logEntry.Message)
				}

				if logEntry.Level.String() != tt.logLevel.String() {
					t.Errorf("Level incorect, expected '%s' got '%s'", tt.logLevel.String(), logEntry.Level.String())
				}

				spans := mt.FinishedSpans()
				if tt.wantTrace {
					if len(spans) != 1 {
						t.Fatalf("expected 1 span, got %d", len(spans))
					}

					if logEntry.ContextMap()["dd.trace_id"] != tt.expectedTraceId {
						t.Errorf("Message dd.trace_id incorect, expected '%d' got '%d'", tt.expectedTraceId, logEntry.ContextMap()["dd.trace_id"])
					}

					if logEntry.ContextMap()["dd.span_id"] != tt.expectedSpanId {
						t.Errorf("Message dd.span_id incorect, expected '%d' got '%d'", tt.expectedSpanId, logEntry.ContextMap()["dd.span_id"])
					}
				}
			}
		})
	}
}

func TestFatalLogFunction(t *testing.T) {
	captureLogger, logsCollector := setupLogsCapture()
	logger := Logger{
		internalLoggger: captureLogger,
	}

	var panicked interface{}
	func() {
		defer func() {
			panicked = recover()
		}()
		logger.Fatal("error text")
		t.Error("logger.Fatal did not exit")
	}()

	if panicked == nil {
		t.Fatal("expected panic on logger.Fatal")
	}

	if len(logsCollector.All()) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logsCollector.All()))
	}

	logEntry := logsCollector.All()[0]
	if logEntry.Message != "error text" {
		t.Errorf("Message incorect, expected '%s' got '%s'", "error text", logEntry.Message)
	}
}

// erroringSyncer is a WriteSyncer that always returns an error on Sync
type erroringSyncer struct{}

func (es *erroringSyncer) Write(p []byte) (n int, err error) {
	return len(p), nil // Write succeeds
}

func (es *erroringSyncer) Sync() error {
	return errors.New("sync error") // Sync always fails
}

func TestSync(t *testing.T) {
	t.Run("successful sync", func(t *testing.T) {
		captureLogger, _ := setupLogsCapture()
		logger := Logger{
			internalLoggger: captureLogger,
		}

		// Should not panic
		logger.Sync()
	})

	t.Run("sync error", func(t *testing.T) {
		// Create a logger with a core that will return an error on sync
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(zapcore.EncoderConfig{}),
			zapcore.AddSync(&erroringSyncer{}),
			zapcore.InfoLevel,
		)
		logger := Logger{
			internalLoggger: zap.New(core).Sugar().WithOptions(zap.WithFatalHook(zapcore.WriteThenPanic)),
		}

		var panicked interface{}
		func() {
			defer func() {
				panicked = recover()
			}()
			logger.Sync()
			t.Error("logger.Sync() did not exit on error")
		}()

		if panicked == nil {
			t.Fatal("expected panic on logger.Sync() error")
		}
	})
}
