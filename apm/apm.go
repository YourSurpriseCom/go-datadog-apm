package apm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"net/http"

	"github.com/YourSurpriseCom/go-datadog-apm/logger"
	"github.com/go-chi/chi/v5"
	sqltrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/database/sql"
	chitrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/go-chi/chi.v5"
	httptrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/net/http"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type Apm struct {
	Logger *logger.Logger
}

type ApmOption func(*Apm)

func WithLogger(logger *logger.Logger) ApmOption {
	return func(apm *Apm) {
		apm.Logger = logger
	}
}

func NewApm(options ...ApmOption) Apm {
	apm := Apm{}

	for _, option := range options {
		option(&apm)
	}

	if apm.Logger == nil {
		logger := logger.NewLogger()
		apm.Logger = &logger
	}

	return apm
}

func (apm Apm) StartSpanFromContext(ctx context.Context, name string) (ddtrace.Span, context.Context) {
	return tracer.StartSpanFromContext(ctx, name)
}

func (apm Apm) SpanFromContext(ctx context.Context) (ddtrace.Span, bool) {
	return tracer.SpanFromContext(ctx)
}

func (apm Apm) ConfigureOnRouter(router *chi.Mux) {
	router.Use(chitrace.Middleware())
}

func (apm Apm) ConfigureOnHttpClient(client *http.Client) {
	originalClient := client
	*client = *httptrace.WrapClient(originalClient)
}

func (apm Apm) ConfigureOnSQLClient(driverName string, driver driver.Driver, dataSourceName string, opts ...sqltrace.Option) (*sql.DB, error) {
	sqltrace.Register(driverName, driver, opts...)

	return sqltrace.Open(driverName, dataSourceName)
}
