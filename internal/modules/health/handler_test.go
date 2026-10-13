package health

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type envelope[T any] struct {
	Data T `json:"data"`
}

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

type fakeSchema struct {
	err    error
	called *bool
}

func (f fakeSchema) CheckSchema(context.Context) error {
	if f.called != nil {
		*f.called = true
	}
	return f.err
}

func newRouter(p Pinger) *gin.Engine {
	return newRouterWithSchema(p, fakeSchema{})
}

func newRouterWithSchema(p Pinger, s SchemaChecker) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewHandler(p, s, "test-version", slog.New(slog.NewTextHandler(io.Discard, nil)))
	h.now = func() time.Time { return time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC) }
	r := gin.New()
	h.Register(r.Group("/api/v1"))
	return r
}

func do(r http.Handler, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

func TestHealth(t *testing.T) {
	w := do(newRouter(fakePinger{}), "/api/v1/health")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var body envelope[Status]
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	want := Status{Status: "ok", Service: ServiceName, Version: "test-version", Time: time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC)}
	if body.Data != want {
		t.Errorf("body = %+v, want %+v", body.Data, want)
	}
}

func TestHealthDoesNotDependOnDatabase(t *testing.T) {
	w := do(newRouter(fakePinger{err: errors.New("down")}), "/api/v1/health")
	if w.Code != http.StatusOK {
		t.Fatalf("liveness must not fail when the database is down; status = %d", w.Code)
	}
}

func TestReady(t *testing.T) {
	w := do(newRouter(fakePinger{}), "/api/v1/ready")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var body envelope[Readiness]
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Data.Status != "ready" || body.Data.Checks["database"] != "up" || body.Data.Checks["migrations"] != "up" {
		t.Errorf("body = %+v", body.Data)
	}
}

func TestReadyWhenDatabaseDown(t *testing.T) {
	secret := "password authentication failed for user app_user at 10.0.0.5"
	w := do(newRouter(fakePinger{err: errors.New(secret)}), "/api/v1/ready")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "password") || strings.Contains(w.Body.String(), "10.0.0.5") {
		t.Errorf("readiness leaked internal error: %s", w.Body.String())
	}
	var body envelope[Readiness]
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Data.Status != "not_ready" || body.Data.Checks["database"] != "down" || body.Data.Checks["migrations"] != "down" {
		t.Errorf("body = %+v", body.Data)
	}
}

func TestReadySkipsSchemaCheckWhenDatabaseDown(t *testing.T) {
	called := false
	w := do(newRouterWithSchema(fakePinger{err: errors.New("down")}, fakeSchema{called: &called}), "/api/v1/ready")
	if w.Code != http.StatusServiceUnavailable || called {
		t.Fatalf("status = %d, schema check called = %t", w.Code, called)
	}
}

func TestReadyWhenSchemaNotCurrent(t *testing.T) {
	reason := "schema is behind this build (migrations pending): database at 0, build expects 1"
	w := do(newRouterWithSchema(fakePinger{}, fakeSchema{err: errors.New(reason)}), "/api/v1/ready")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "behind") {
		t.Errorf("readiness leaked internal reason: %s", w.Body.String())
	}
	var body envelope[Readiness]
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	want := map[string]string{"database": "up", "migrations": "down"}
	if body.Data.Status != "not_ready" || len(body.Data.Checks) != 2 ||
		body.Data.Checks["database"] != want["database"] || body.Data.Checks["migrations"] != want["migrations"] {
		t.Errorf("body = %+v", body.Data)
	}
}
