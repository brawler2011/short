package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newBenchStore(tb testing.TB) *Store {
	tb.Helper()
	dbPath := filepath.Join(tb.TempDir(), "bench.db")
	store, err := NewStore(dbPath)
	if err != nil {
		tb.Fatalf("NewStore: %v", err)
	}
	return store
}

// BenchmarkRandomCode measures random code generation speed and memory allocations.
func BenchmarkRandomCode(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := randomCode(6)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStore_Create measures inserting short links into SQLite.
func BenchmarkStore_Create(b *testing.B) {
	store := newBenchStore(b)
	defer func() { _ = store.Close() }()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.Create(fmt.Sprintf("https://example.com/page/%d", i))
		if err != nil {
			b.Fatalf("store.Create: %v", err)
		}
	}
}

// BenchmarkStore_Resolve measures reading URLs by short code.
func BenchmarkStore_Resolve(b *testing.B) {
	store := newBenchStore(b)
	defer func() { _ = store.Close() }()

	code, err := store.Create("https://example.com/target-benchmark")
	if err != nil {
		b.Fatalf("store.Create: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rawURL, err := store.Resolve(code)
		if err != nil || rawURL == "" {
			b.Fatalf("store.Resolve failed: %v", err)
		}
	}
}

// BenchmarkStore_Resolve_Parallel measures concurrent URL lookups across goroutines.
func BenchmarkStore_Resolve_Parallel(b *testing.B) {
	store := newBenchStore(b)
	defer func() { _ = store.Close() }()

	code, err := store.Create("https://example.com/target-parallel")
	if err != nil {
		b.Fatalf("store.Create: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rawURL, err := store.Resolve(code)
			if err != nil || rawURL == "" {
				b.Fatalf("store.Resolve: %v", err)
			}
		}
	})
}

// BenchmarkStore_ValidateSession measures session validation speed.
func BenchmarkStore_ValidateSession(b *testing.B) {
	store := newBenchStore(b)
	defer func() { _ = store.Close() }()

	token := "bench-session-token"
	if err := store.CreateSession(token); err != nil {
		b.Fatalf("CreateSession: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		valid, err := store.ValidateSession(token)
		if err != nil || !valid {
			b.Fatalf("ValidateSession: %v", err)
		}
	}
}

// BenchmarkHandler_Shorten benchmarks the full HTTP POST /api/shorten pipeline.
func BenchmarkHandler_Shorten(b *testing.B) {
	h, store := newTestHandler(b)
	defer func() { _ = store.Close() }()

	body, _ := json.Marshal(map[string]string{"url": "https://example.com/bench-http"})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("expected 200, got %d", w.Code)
		}
	}
}

// BenchmarkHandler_Redirect benchmarks HTTP GET /{code} redirect pipeline.
func BenchmarkHandler_Redirect(b *testing.B) {
	h, store := newTestHandler(b)
	defer func() { _ = store.Close() }()

	code, err := store.Create("https://example.com/bench-redirect")
	if err != nil {
		b.Fatalf("store.Create: %v", err)
	}

	reqPath := "/" + code

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, reqPath, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusFound {
			b.Fatalf("expected 302, got %d", w.Code)
		}
	}
}

// BenchmarkHandler_Redirect_Parallel benchmarks concurrent HTTP redirects.
func BenchmarkHandler_Redirect_Parallel(b *testing.B) {
	h, store := newTestHandler(b)
	defer func() { _ = store.Close() }()

	code, err := store.Create("https://example.com/bench-redirect-parallel")
	if err != nil {
		b.Fatalf("store.Create: %v", err)
	}

	reqPath := "/" + code

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, reqPath, nil)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != http.StatusFound {
				b.Fatalf("expected 302, got %d", w.Code)
			}
		}
	})
}

// TestPerformance_HighConcurrency_CreateAndResolve tests concurrent URL shortening and resolving.
func TestPerformance_HighConcurrency_CreateAndResolve(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	h, store := newTestHandler(t)
	defer func() { _ = store.Close() }()

	const (
		numWorkers   = 20
		opsPerWorker = 50
	)
	totalOps := numWorkers * opsPerWorker

	var wg sync.WaitGroup
	var successCount atomic.Int64
	latencies := make([]time.Duration, 0, totalOps)
	var latenciesMu sync.Mutex

	start := time.Now()

	for worker := 0; worker < numWorkers; worker++ {
		wg.Add(1)
		go func(wID int) {
			defer wg.Done()
			for j := 0; j < opsPerWorker; j++ {
				opStart := time.Now()

				// 1. Create short link via API
				targetURL := fmt.Sprintf("https://example.com/worker/%d/item/%d", wID, j)
				body, _ := json.Marshal(map[string]string{"url": targetURL})
				req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, req)

				if rec.Code != http.StatusOK {
					t.Errorf("worker %d failed to shorten: %d", wID, rec.Code)
					continue
				}

				var resp map[string]string
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Errorf("worker %d invalid json: %v", wID, err)
					continue
				}
				code := resp["code"]

				// 2. Resolve short link via Redirect
				getReq := httptest.NewRequest(http.MethodGet, "/"+code, nil)
				getRec := httptest.NewRecorder()
				h.ServeHTTP(getRec, getReq)

				if getRec.Code != http.StatusFound {
					t.Errorf("worker %d failed to resolve: %d", wID, getRec.Code)
					continue
				}

				elapsed := time.Since(opStart)
				latenciesMu.Lock()
				latencies = append(latencies, elapsed)
				latenciesMu.Unlock()

				successCount.Add(1)
			}
		}(worker)
	}

	wg.Wait()
	totalDuration := time.Since(start)

	if successCount.Load() != int64(totalOps) {
		t.Fatalf("expected %d successful ops, got %d", totalOps, successCount.Load())
	}

	rps := float64(totalOps) / totalDuration.Seconds()

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	p50 := latencies[len(latencies)/2]
	p95 := latencies[int(float64(len(latencies))*0.95)]
	p99 := latencies[int(float64(len(latencies))*0.99)]

	t.Logf("Performance Report (High Concurrency Create & Resolve):")
	t.Logf("  Workers:       %d", numWorkers)
	t.Logf("  Total Ops:     %d", totalOps)
	t.Logf("  Total Time:    %v", totalDuration)
	t.Logf("  Throughput:    %.2f ops/sec (RPS)", rps)
	t.Logf("  P50 Latency:   %v", p50)
	t.Logf("  P95 Latency:   %v", p95)
	t.Logf("  P99 Latency:   %v", p99)
}

// TestPerformance_Store_HeavyConcurrentReads tests concurrent read throughput on Store.
func TestPerformance_Store_HeavyConcurrentReads(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	store := newBenchStore(t)
	defer func() { _ = store.Close() }()

	const numLinks = 50
	codes := make([]string, numLinks)
	for i := 0; i < numLinks; i++ {
		c, err := store.Create(fmt.Sprintf("https://example.com/link-%d", i))
		if err != nil {
			t.Fatalf("store.Create: %v", err)
		}
		codes[i] = c
	}

	const (
		numReaders     = 20
		readsPerReader = 100
	)
	totalReads := numReaders * readsPerReader

	var wg sync.WaitGroup
	var successReads atomic.Int64
	start := time.Now()

	for r := 0; r < numReaders; r++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for i := 0; i < readsPerReader; i++ {
				code := codes[(readerID+i)%numLinks]
				target, err := store.Resolve(code)
				if err != nil || target == "" {
					t.Errorf("read failed for %s: %v", code, err)
					return
				}
				successReads.Add(1)
			}
		}(r)
	}

	wg.Wait()
	dur := time.Since(start)
	rps := float64(totalReads) / dur.Seconds()

	if successReads.Load() != int64(totalReads) {
		t.Fatalf("expected %d successful reads, got %d", totalReads, successReads.Load())
	}

	t.Logf("Performance Report (Concurrent Store Reads):")
	t.Logf("  Readers:       %d", numReaders)
	t.Logf("  Total Reads:   %d", totalReads)
	t.Logf("  Total Time:    %v", dur)
	t.Logf("  Read RPS:      %.2f queries/sec", rps)
}
