package apm

import (
	"context"
	"database/sql/driver"
	"net/http"
	"reflect"
	"testing"
	"time"

	sqltrace "github.com/DataDog/dd-trace-go/contrib/database/sql/v2"
	chitrace "github.com/DataDog/dd-trace-go/contrib/go-chi/chi.v5/v2"
	httptrace "github.com/DataDog/dd-trace-go/contrib/net/http/v2"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/mocktracer"
	"github.com/YourSurpriseCom/go-datadog-apm/v2/logger"
	"github.com/go-chi/chi/v5"
)

func TestNewApm(t *testing.T) {
	apm := NewApm()

	if reflect.TypeOf(apm.Logger) != reflect.TypeOf(&logger.Logger{}) {
		t.Errorf("Logger type incorrect, expected '%s' got '%s'", reflect.TypeOf(logger.Logger{}), reflect.TypeOf(apm.Logger))
	}

	loggerName := "apm-logger"
	myLogger := logger.NewLogger(logger.WithName(loggerName))
	apm = NewApm(WithLogger(myLogger))

	if apm.Logger.Name() != "apm-logger" {
		t.Errorf("Apm logger name incorrect, expected '%s' got '%s'", loggerName, apm.Logger.Name())
	}
}

func TestStartSpanFromContext(t *testing.T) {
	mt := mocktracer.Start()
	defer mt.Stop()
	ctx := context.Background()

	apm := NewApm()
	span, spanContext := apm.StartSpanFromContext(ctx, "test new span")
	span.Finish()

	if spanContext == ctx {
		t.Fatal("expected 'spanContext' not to be the same as 'ctx'")
	}

	spans := mt.FinishedSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
}

func TestSpanFromContext(t *testing.T) {
	mt := mocktracer.Start()
	defer mt.Stop()

	apm := NewApm()

	t.Run("when no span exists in context", func(t *testing.T) {
		ctx := context.Background()
		_, exists := apm.SpanFromContext(ctx)

		if exists {
			t.Error("expected exists to be false when no span in context")
		}
	})

	t.Run("when span exists in context", func(t *testing.T) {
		ctx := context.Background()
		originalSpan, spanCtx := apm.StartSpanFromContext(ctx, "test span")
		defer originalSpan.Finish()

		retrievedSpan, exists := apm.SpanFromContext(spanCtx)

		if !exists {
			t.Error("expected exists to be true when span exists in context")
		}
		if retrievedSpan.Context().TraceID() != originalSpan.Context().TraceID() ||
			retrievedSpan.Context().SpanID() != originalSpan.Context().SpanID() {
			t.Error("retrieved span does not match original span")
		}
	})
}

func TestConfigureOnRouter(t *testing.T) {
	mt := mocktracer.Start()
	defer mt.Stop()

	apm := NewApm()
	router := chi.NewRouter()

	// Configure APM on router
	apm.ConfigureOnRouter(router)

	// Get the middlewares
	middlewares := router.Middlewares()

	if len(middlewares) == 0 {
		t.Fatal("No middleware was added to the router")
	}

	// Get the last added middleware
	lastMiddleware := middlewares[len(middlewares)-1]

	// Create a reference middleware for type comparison
	refMiddleware := chitrace.Middleware()

	// Compare the types
	actualType := reflect.TypeOf(lastMiddleware)
	expectedType := reflect.TypeOf(refMiddleware)

	if actualType != expectedType {
		t.Errorf("Wrong middleware type added. Expected %v, got %v", expectedType, actualType)
	}
}

func TestConfigureOnHttpClient(t *testing.T) {
	mt := mocktracer.Start()
	defer mt.Stop()

	apm := NewApm()

	// Create a client with custom settings
	originalClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Configure APM on the client
	apm.ConfigureOnHttpClient(originalClient)

	// Create a reference wrapped client for type comparison
	refClient := httptrace.WrapClient(&http.Client{})

	// Get the underlying type of the wrapped client
	wrappedClientType := reflect.TypeOf(refClient).Elem()
	actualClientType := reflect.TypeOf(*originalClient)

	if actualClientType != wrappedClientType {
		t.Errorf("Wrong client type after wrapping. Expected %v, got %v", wrappedClientType, actualClientType)
	}

	// Verify original settings are preserved
	if originalClient.Timeout != 5*time.Second {
		t.Errorf("Client settings were not preserved. Expected timeout 5s, got %v", originalClient.Timeout)
	}
}

func TestConfigureOnSQLClient(t *testing.T) {
	mt := mocktracer.Start()
	defer mt.Stop()

	apm := NewApm()
	driverName := "mock-sql-driver"
	dataSourceName := "mock-connection-string"

	// Configure SQL client with tracing
	opts := []sqltrace.Option{
		sqltrace.WithService("test-service"),
		sqltrace.WithAnalytics(true),
	}
	db, err := apm.ConfigureOnSQLClient(driverName, &mockDriver{}, dataSourceName, opts...)
	if err != nil {
		t.Fatalf("Failed to configure SQL client: %v", err)
	}

	// Verify db connection was created
	if db == nil {
		t.Fatal("Expected db connection to be created, got nil")
	}

	err = db.Ping()
	if err != nil {
		t.Fatalf("Unexpected error while running db.Ping: %v", err)
	}

	spans := mt.FinishedSpans()
	if len(spans) == 2 {
		pingSpan := spans[1]
		if pingSpan.Tag("sql.query_type") != "Ping" {
			t.Fatalf("expected span name to be '%s', got '%s'", "Ping", pingSpan.Tag("sql.query_type"))
		}
	} else {
		t.Fatalf("expected 2 spans, got %d", len(spans))

	}

}

func TestConfigureOnSQLXClient(t *testing.T) {
	mt := mocktracer.Start()
	defer mt.Stop()

	apm := NewApm()
	driverName := "mock-sqlx-driver"
	dataSourceName := "mock-connection-string"

	// Configure SQL client with tracing
	opts := []sqltrace.Option{
		sqltrace.WithService("test-service"),
		sqltrace.WithAnalytics(true),
	}
	db, err := apm.ConfigureOnSQLXClient(driverName, &mockDriver{}, dataSourceName, opts...)
	if err != nil {
		t.Fatalf("Failed to configure SQL client: %v", err)
	}

	// Verify db connection was created
	if db == nil {
		t.Fatal("Expected db connection to be created, got nil")
	}

	err = db.Ping()
	if err != nil {
		t.Fatalf("Unexpected error while running db.Ping: %v", err)
	}

	spans := mt.FinishedSpans()
	if len(spans) == 2 {
		pingSpan := spans[1]
		if pingSpan.Tag("sql.query_type") != "Ping" {
			t.Fatalf("expected span name to be '%s', got '%s'", "Ping", pingSpan.Tag("sql.query_type"))
		}
	} else {
		t.Fatalf("expected 2 spans, got %d", len(spans))

	}

}

// mockDriver implements database/sql/driver.Driver interface
type mockDriver struct{}

func (d *mockDriver) Open(name string) (driver.Conn, error) {
	return &mockConn{}, nil
}

// mockConn implements database/sql/driver.Conn interface
type mockConn struct{}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) {
	return &mockStmt{}, nil
}

func (c *mockConn) Close() error {
	return nil
}

func (c *mockConn) Begin() (driver.Tx, error) {
	return &mockTx{}, nil
}

// mockTx implements database/sql/driver.Tx
type mockTx struct{}

func (tx *mockTx) Commit() error {
	return nil
}

func (tx *mockTx) Rollback() error {
	return nil
}

// mockStmt implements database/sql/driver.Stmt
type mockStmt struct{}

func (s *mockStmt) Close() error {
	return nil
}

func (s *mockStmt) NumInput() int {
	return 0
}

func (s *mockStmt) Exec(args []driver.Value) (driver.Result, error) {
	return &mockResult{}, nil
}

func (s *mockStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &mockRows{}, nil
}

// mockResult implements database/sql/driver.Result
type mockResult struct{}

func (r *mockResult) LastInsertId() (int64, error) {
	return 0, nil
}

func (r *mockResult) RowsAffected() (int64, error) {
	return 0, nil
}

// mockRows implements database/sql/driver.Rows
type mockRows struct{}

func (r *mockRows) Columns() []string {
	return []string{}
}

func (r *mockRows) Close() error {
	return nil
}

func (r *mockRows) Next(dest []driver.Value) error {
	return nil
}
