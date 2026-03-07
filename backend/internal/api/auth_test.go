package api

import (
	"testing"
	"time"
)

func TestGenerateAndValidateToken(t *testing.T) {
	secret := "test-secret-key-for-testing-only"
	userID := int64(42)

	token, err := generateAccessToken(userID, secret, 15*time.Minute)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := validateToken(token, secret)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected userID %d, got %d", userID, claims.UserID)
	}
}

func TestValidateToken_Expired(t *testing.T) {
	secret := "test-secret"
	token, _ := generateAccessToken(1, secret, -1*time.Minute)

	_, err := validateToken(token, secret)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestHashAndCheckPassword(t *testing.T) {
	password := "mysecretpassword"
	hash, err := hashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash: %v", err)
	}

	if !checkPassword(password, hash) {
		t.Error("password should match hash")
	}
	if checkPassword("wrongpassword", hash) {
		t.Error("wrong password should not match hash")
	}
}
