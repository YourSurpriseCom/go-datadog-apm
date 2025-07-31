package apm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"net/http"

	sqltrace "github.com/DataDog/dd-trace-go/contrib/database/sql/v2"
	chitrace "github.com/DataDog/dd-trace-go/contrib/go-chi/chi.v5/v2"
	gormtrace "github.com/DataDog/dd-trace-go/contrib/gorm.io/gorm.v1/v2"
	sqlxtrace "github.com/DataDog/dd-trace-go/contrib/jmoiron/sqlx/v2"
	httptrace "github.com/DataDog/dd-trace-go/contrib/net/http/v2"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/YourSurpriseCom/go-datadog-apm/v2/logger"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

type Apm struct {
	Logger *logger.Logger
}

type ApmOption func(*Apm)

func WithLogger(logger logger.Logger) ApmOption {
	return func(apm *Apm) {
		apm.Logger = &logger
	}
}

// NewApm creates a new Apm instance with the provided options.
// Example:
//
//	myLogger := &logger.Logger{}
//	apm := NewApm(WithLogger(myLogger))
//	apm.Logger.Info(context.Background(), "Hello, world!")
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

func (apm Apm) StartSpanFromContext(ctx context.Context, name string) (*tracer.Span, context.Context) {
	return tracer.StartSpanFromContext(ctx, name)
}

func (apm Apm) SpanFromContext(ctx context.Context) (*tracer.Span, bool) {
	return tracer.SpanFromContext(ctx)
}

func (apm Apm) ConfigureOnRouter(router *chi.Mux, opts ...chitrace.Option) {
	router.Use(chitrace.Middleware(opts...))
}

func (apm Apm) ConfigureOnHttpClient(client *http.Client, opts ...httptrace.RoundTripperOption) *http.Client {
	originalClient := client
	*client = *httptrace.WrapClient(originalClient, opts...)
	return client
}

func (apm Apm) ConfigureOnSQLClient(driverName string, driver driver.Driver, dataSourceName string, opts ...sqltrace.Option) (*sql.DB, error) {
	sqltrace.Register(driverName, driver, opts...)

	return sqltrace.Open(driverName, dataSourceName)
}

func (apm Apm) ConfigureOnSQLXClient(driverName string, driver driver.Driver, dataSourceName string, opts ...sqltrace.Option) (*sqlx.DB, error) {
	sqltrace.Register(driverName, driver, opts...)

	return sqlxtrace.Open(driverName, dataSourceName)
}

func (apm Apm) ConfigureOnGormMySQLClient(dialector gorm.Dialector, cfg *gorm.Config, opts ...gormtrace.Option) (*gorm.DB, error) {
	sqltrace.Register("mysql", &mysql.MySQLDriver{})

	return gormtrace.Open(dialector, cfg, opts...)
}
