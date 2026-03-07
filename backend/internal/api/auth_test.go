package api

import (
	"testing"
	"time"
)

func TestGenerateAndValidateToken(t *testing.T) {
	secret := "test-secret-key-for-testing-only"
	userID := int64(42)

	token, err := generateToken(userID, secret, 15*time.Minute, "access")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := validateToken(token, secret, "access")
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected userID %d, got %d", userID, claims.UserID)
	}
	if claims.TokenType != "access" {
		t.Errorf("expected token type access, got %s", claims.TokenType)
	}
}

func TestValidateToken_Expired(t *testing.T) {
	secret := "test-secret"
	token, _ := generateToken(1, secret, -1*time.Minute, "access")

	_, err := validateToken(token, secret, "access")
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestValidateToken_WrongType(t *testing.T) {
	secret := "test-secret"
	token, _ := generateToken(1, secret, 15*time.Minute, "refresh")

	_, err := validateToken(token, secret, "access")
	if err == nil {
		t.Error("expected error when using refresh token as access token")
	}
}

func TestRefreshTokenCannotBeUsedAsAccess(t *testing.T) {
	secret := "test-secret"
	refreshToken, _ := generateToken(1, secret, 7*24*time.Hour, "refresh")

	_, err := validateToken(refreshToken, secret, "access")
	if err == nil {
		t.Error("refresh token should not validate as access token")
	}

	accessToken, _ := generateToken(1, secret, 24*time.Hour, "access")
	_, err = validateToken(accessToken, secret, "refresh")
	if err == nil {
		t.Error("access token should not validate as refresh token")
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
