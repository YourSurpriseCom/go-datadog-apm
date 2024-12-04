package apm

import (
	"context"
	"reflect"
	"testing"

	"github.com/YourSurpriseCom/go-datadog-apm/logger"
	"github.com/go-chi/chi/v5"
	chitrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/go-chi/chi.v5"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/mocktracer"
)

func TestNewApm(t *testing.T) {
	apm := NewApm()

	if reflect.TypeOf(apm.Logger) != reflect.TypeOf(logger.Logger{}) {
		t.Errorf("Logger type incorect, expected '%s' got '%s'", reflect.TypeOf(logger.Logger{}), reflect.TypeOf(apm.Logger))
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
		if retrievedSpan != originalSpan {
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
