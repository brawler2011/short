package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func newTestHandler(tb testing.TB) (*Handler, *Store) {
	tb.Helper()
	dbPath := filepath.Join(tb.TempDir(), "test.db")
	store, err := NewStore(dbPath)
	if err != nil {
		tb.Fatalf("NewStore: %v", err)
	}

	hub := NewHub()
	mockFS := fstest.MapFS{
		"index.html":  &fstest.MapFile{Data: []byte("<!DOCTYPE html><html><body>Test</body></html>")},
		"favicon.svg": &fstest.MapFile{Data: []byte("<svg>test</svg>")},
		"favicon.ico": &fstest.MapFile{Data: []byte("ico-data")},
	}
	h := NewHandler(store, hub, "https://short.steins.ru", mockFS)
	return h, store
}

func TestHandler_Shorten(t *testing.T) {
	h, store := newTestHandler(t)
	defer func() { _ = store.Close() }()

	// 1. Valid URL
	body, _ := json.Marshal(map[string]string{"url": "https://google.com"})
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if res["short_url"] == "" || res["code"] == "" {
		t.Fatalf("invalid shorten response: %v", res)
	}

	// 2. Invalid URL
	badBody, _ := json.Marshal(map[string]string{"url": "not-valid-url"})
	badReq := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(badBody))
	badW := httptest.NewRecorder()
	h.ServeHTTP(badW, badReq)
	if badW.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", badW.Code)
	}
}

func TestHandler_Redirect(t *testing.T) {
	h, store := newTestHandler(t)
	defer func() { _ = store.Close() }()

	code, err := store.Create("https://example.com/target")
	if err != nil {
		t.Fatalf("store.Create: %v", err)
	}

	// 1. Successful redirect
	req := httptest.NewRequest(http.MethodGet, "/"+code, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "https://example.com/target" {
		t.Fatalf("expected redirect to https://example.com/target, got %s", loc)
	}

	// 2. Not found
	req404 := httptest.NewRequest(http.MethodGet, "/unknowncode", nil)
	w404 := httptest.NewRecorder()
	h.ServeHTTP(w404, req404)
	if w404.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w404.Code)
	}
}

func TestHandler_SessionPush_InvalidSession(t *testing.T) {
	h, store := newTestHandler(t)
	defer func() { _ = store.Close() }()

	token := "valid-token"
	if err := store.CreateSession(token); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// Push with non-existent session token
	body, _ := json.Marshal(map[string]string{
		"session_token": "non-existent-token",
		"url":           "https://example.com",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/session/push", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestHandler_Favicon(t *testing.T) {
	h, store := newTestHandler(t)
	defer func() { _ = store.Close() }()

	for _, path := range []string{"/favicon.svg", "/favicon.ico"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d", path, w.Code)
		}
	}
}
