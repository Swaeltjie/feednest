package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/feednest/backend/internal/apiutil"
	"github.com/feednest/backend/internal/models"
	"github.com/feednest/backend/internal/store"
)

var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+$`)

type Claims struct {
	UserID    int64  `json:"user_id"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

type AuthHandler struct {
	store     *store.Queries
	jwtSecret string
}

func NewAuthHandler(store *store.Queries, jwtSecret string) *AuthHandler {
	return &AuthHandler{store: store, jwtSecret: jwtSecret}
}

func generateToken(userID int64, secret string, duration time.Duration, tokenType string) (string, error) {
	claims := Claims{
		UserID:    userID,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func validateToken(tokenStr string, secret string, expectedType string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.TokenType != expectedType {
		return nil, fmt.Errorf("expected %s token, got %s", expectedType, claims.TokenType)
	}
	return claims, nil
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func checkPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		http.Error(w, `{"error":"username, email, and password are required"}`, http.StatusBadRequest)
		return
	}

	// Normalize username and email to lowercase to prevent case-based duplicates
	req.Username = strings.ToLower(strings.TrimSpace(req.Username))
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if utf8.RuneCountInString(req.Username) > 64 || !usernameRe.MatchString(req.Username) {
		http.Error(w, `{"error":"username must be 1-64 chars, alphanumeric/underscore/dash/dot only"}`, http.StatusBadRequest)
		return
	}

	if _, err := mail.ParseAddress(req.Email); err != nil || len(req.Email) > 254 {
		http.Error(w, `{"error":"invalid email address"}`, http.StatusBadRequest)
		return
	}

	if len(req.Password) < 8 {
		http.Error(w, `{"error":"password must be at least 8 characters"}`, http.StatusBadRequest)
		return
	}
	if len(req.Password) > 72 {
		http.Error(w, `{"error":"password must not exceed 72 characters"}`, http.StatusBadRequest)
		return
	}

	hash, err := hashPassword(req.Password)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	user, err := h.store.CreateUser(req.Username, req.Email, hash)
	if err != nil {
		http.Error(w, `{"error":"username or email already taken"}`, http.StatusConflict)
		return
	}

	accessToken, err := generateToken(user.ID, h.jwtSecret, 24*time.Hour, "access")
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	refreshToken, err := generateToken(user.ID, h.jwtSecret, 7*24*time.Hour, "refresh")
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	resp := models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *user,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if len(req.Password) > 72 {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	// Normalize username to match registration
	req.Username = strings.ToLower(strings.TrimSpace(req.Username))

	user, err := h.store.GetUserByUsername(req.Username)
	if err != nil {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	if !checkPassword(req.Password, user.PasswordHash) {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	accessToken, err := generateToken(user.ID, h.jwtSecret, 24*time.Hour, "access")
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}
	refreshToken, err := generateToken(user.ID, h.jwtSecret, 7*24*time.Hour, "refresh")
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	resp := models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	claims, err := validateToken(body.RefreshToken, h.jwtSecret, "refresh")
	if err != nil {
		http.Error(w, `{"error":"invalid refresh token"}`, http.StatusUnauthorized)
		return
	}

	accessToken, err := generateToken(claims.UserID, h.jwtSecret, 24*time.Hour, "access")
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"access_token": accessToken})
}

func (h *AuthHandler) UserCount(w http.ResponseWriter, r *http.Request) {
	count, err := h.store.UserCount()
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"count": count, "has_users": count > 0})
}

// ExtractUserID extracts the authenticated user ID from the request context.
func ExtractUserID(r *http.Request) int64 {
	return apiutil.ExtractUserID(r)
}
