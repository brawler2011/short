package main

import (
	"path/filepath"
	"testing"
)

func TestStore_CreateAndResolve(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer func() { _ = s.Close() }()

	originalURL := "https://github.com/brawler2011/short"
	code, err := s.Create(originalURL)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if len(code) != 6 {
		t.Errorf("expected code length 6, got %d (%s)", len(code), code)
	}

	resolved, err := s.Resolve(code)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if resolved != originalURL {
		t.Errorf("expected %s, got %s", originalURL, resolved)
	}

	// Non-existent code
	nonExistent, err := s.Resolve("nonexist")
	if err != nil {
		t.Fatalf("Resolve non-existent failed: %v", err)
	}
	if nonExistent != "" {
		t.Errorf("expected empty string for non-existent code, got %s", nonExistent)
	}
}

func TestStore_Session(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer func() { _ = s.Close() }()

	token := "session-token-123"

	err = s.CreateSession(token)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	valid, err := s.ValidateSession(token)
	if err != nil {
		t.Fatalf("ValidateSession failed: %v", err)
	}
	if !valid {
		t.Errorf("expected session to be valid")
	}

	// Unknown token
	invalidToken, err := s.ValidateSession("unknown-token")
	if err != nil {
		t.Fatalf("ValidateSession unknown token failed: %v", err)
	}
	if invalidToken {
		t.Errorf("expected unknown token to be invalid")
	}

	// Delete session
	if err := s.DeleteSession(token); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	validAfterDelete, err := s.ValidateSession(token)
	if err != nil {
		t.Fatalf("ValidateSession after delete failed: %v", err)
	}
	if validAfterDelete {
		t.Errorf("expected deleted session to be invalid")
	}
}
