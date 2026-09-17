package main

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Handler struct {
	store   *Store
	hub     *Hub
	baseURL string
	static  fs.FS
}

func NewHandler(store *Store, hub *Hub, baseURL string, static fs.FS) *Handler {
	return &Handler{
		store:   store,
		hub:     hub,
		baseURL: strings.TrimRight(baseURL, "/"),
		static:  static,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/ws" && r.Method == http.MethodGet:
		h.handleWS(w, r)
	case path == "/api/shorten" && r.Method == http.MethodPost:
		h.handleShorten(w, r)
	case path == "/api/session/push" && r.Method == http.MethodPost:
		h.handlePush(w, r)
	case strings.HasPrefix(path, "/assets/"):
		h.serveStatic(w, r)
	case path == "/favicon.ico" || path == "/favicon.svg":
		h.serveStatic(w, r)
	case strings.HasPrefix(path, "/session/"):
		h.serveIndex(w, r)
	case path == "/" || path == "/index.html":
		h.serveIndex(w, r)
	case isSingleSegment(path):
		h.handleRedirect(w, r, path[1:])
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) serveIndex(w http.ResponseWriter, r *http.Request) {
	content, err := fs.ReadFile(h.static, "index.html")
	if err != nil {
		http.Error(w, "frontend not built", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(content)
}

func (h *Handler) serveStatic(w http.ResponseWriter, r *http.Request) {
	http.FileServer(http.FS(h.static)).ServeHTTP(w, r)
}

func (h *Handler) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}

	token := uuid.New().String()
	if err := h.store.CreateSession(token); err != nil {
		log.Printf("create session error: %v", err)
		_ = conn.Close()
		return
	}

	h.hub.Register(token, conn)
	defer func() {
		h.hub.Unregister(token)
		_ = h.store.DeleteSession(token)
		_ = conn.Close()
	}()

	sessionURL := h.baseURL + "/session/" + token
	err = sendJSON(conn, WSMessage{
		Type:       "session_created",
		Token:      token,
		SessionURL: sessionURL,
	})
	if err != nil {
		log.Printf("ws send session_created: %v", err)
		return
	}

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (h *Handler) handleShorten(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if !validURL(req.URL) {
		jsonError(w, "invalid URL: must start with http:// or https://", http.StatusBadRequest)
		return
	}

	code, err := h.store.Create(req.URL)
	if err != nil {
		log.Printf("store.Create: %v", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"short_url": h.baseURL + "/" + code,
		"code":      code,
	})
}

func (h *Handler) handlePush(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionToken string `json:"session_token"`
		URL          string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.SessionToken == "" || req.URL == "" {
		jsonError(w, "missing required fields", http.StatusBadRequest)
		return
	}
	if !validURL(req.URL) {
		jsonError(w, "invalid URL: must start with http:// or https://", http.StatusBadRequest)
		return
	}

	valid, err := h.store.ValidateSession(req.SessionToken)
	if err != nil {
		log.Printf("validate session: %v", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !valid {
		jsonError(w, "invalid session or session expired", http.StatusUnauthorized)
		return
	}

	code, err := h.store.Create(req.URL)
	if err != nil {
		log.Printf("store.Create: %v", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	shortURL := h.baseURL + "/" + code

	if err := h.hub.Push(req.SessionToken, WSMessage{
		Type:        "link_received",
		ShortURL:    shortURL,
		OriginalURL: req.URL,
	}); err != nil {
		log.Printf("hub.Push: %v", err)
		jsonError(w, "computer not connected", http.StatusGone)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"short_url":    shortURL,
		"original_url": req.URL,
		"code":         code,
	})
}

func (h *Handler) handleRedirect(w http.ResponseWriter, r *http.Request, code string) {
	rawURL, err := h.store.Resolve(code)
	if err != nil {
		log.Printf("store.Resolve: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if rawURL == "" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, rawURL, http.StatusFound)
}

func validURL(raw string) bool {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func isSingleSegment(path string) bool {
	if len(path) <= 1 || strings.Contains(path[1:], "/") {
		return false
	}
	// Avoid matching common root files if requested
	seg := path[1:]
	if seg == "favicon.ico" || seg == "favicon.svg" || seg == "robots.txt" {
		return false
	}
	return true
}

func sendJSON(conn *websocket.Conn, msg WSMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, data)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
