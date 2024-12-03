package apm

import (
	"context"

	"github.com/YourSurpriseCom/go-datadog-apm/logger"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type Apm struct {
	Logger logger.Logger
}

func NewApm() Apm {
	logger := logger.NewLogger()
	return Apm{
		Logger: logger,
	}
}

func (apm Apm) StartSpanFromContext(ctx context.Context, name string) (ddtrace.Span, context.Context) {

	return tracer.StartSpanFromContext(ctx, name)
}
