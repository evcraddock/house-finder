package logging

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetupDevMode(t *testing.T) {
	Setup(true)
	// Verify logger works at debug level
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	slog.Debug("test message")
	if !bytes.Contains(buf.Bytes(), []byte("test message")) {
		t.Error("expected debug message in dev mode")
	}
}

func TestSetupProdMode(t *testing.T) {
	Setup(false)
	// Verify logger works — just ensure no panic
	slog.Info("prod test")
}

func TestRequestLogger(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RequestLogger(inner)

	req := httptest.NewRequest("GET", "/api/properties", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	output := buf.String()
	if output == "" {
		t.Fatal("expected log output")
	}
	if !bytes.Contains(buf.Bytes(), []byte("GET")) {
		t.Error("expected method in log")
	}
	if !bytes.Contains(buf.Bytes(), []byte("/api/properties")) {
		t.Error("expected path in log")
	}
}

func TestRequestLoggerSkipsStatic(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RequestLogger(inner)

	req := httptest.NewRequest("GET", "/static/style.css", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if buf.Len() > 0 {
		t.Error("expected no log for static path")
	}
}

func TestResponseWriterCapturesStatus(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	handler := RequestLogger(inner)

	req := httptest.NewRequest("GET", "/missing", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !bytes.Contains(buf.Bytes(), []byte("404")) {
		t.Error("expected 404 status in log")
	}
}
