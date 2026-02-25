package auth

import (
	"testing"
	"time"
)

func TestTokenManagerGenerateParse(t *testing.T) {
	tm, err := NewTokenManager([]byte("test-secret"))
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	now := time.Unix(1700000000, 0).UTC()
	token, err := tm.Generate(69, now)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	userID, err := tm.Parse(token, now.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if userID != 69 {
		t.Fatalf("userID = %d, want 69", userID)
	}
}

func TestTokenManagerParseExpired(t *testing.T) {
	tm, err := NewTokenManager([]byte("test-secret"))
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	now := time.Unix(1700000000, 0).UTC()
	token, err := tm.Generate(69, now)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	_, err = tm.Parse(token, now.Add(25*time.Hour))
	if err != ErrTokenExpired {
		t.Fatalf("Parse() error = %v, want %v", err, ErrTokenExpired)
	}
}

func TestTokenManagerParseInvalidSignature(t *testing.T) {
	tm, err := NewTokenManager([]byte("test-secret"))
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	now := time.Unix(1700000000, 0).UTC()
	token, err := tm.Generate(69, now)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	token += "tampered"
	_, err = tm.Parse(token, now)
	if err != ErrInvalidToken {
		t.Fatalf("Parse() error = %v, want %v", err, ErrInvalidToken)
	}
}
