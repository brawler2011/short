package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
)

//go:embed frontend/dist
var embeddedFrontend embed.FS

func main() {
	baseURL := getenv("BASE_URL", "http://localhost:8080")
	port := getenv("PORT", "8080")
	dbPath := getenv("DB_PATH", "./data/short.db")

	if err := os.MkdirAll("./data", 0o755); err != nil {
		log.Fatalf("mkdir data: %v", err)
	}

	store, err := NewStore(dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer func() { _ = store.Close() }()

	hub := NewHub()

	fsys, err := fs.Sub(embeddedFrontend, "frontend/dist")
	if err != nil {
		log.Fatalf("fs.Sub: %v", err)
	}

	handler := NewHandler(store, hub, baseURL, fsys)

	log.Printf("starting on :%s  base=%s", port, baseURL)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
