package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/feednest/backend/internal/apiutil"
	"github.com/feednest/backend/internal/models"
	"github.com/feednest/backend/internal/store"
)

type Claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

type AuthHandler struct {
	store     *store.Queries
	jwtSecret string
}

func NewAuthHandler(store *store.Queries, jwtSecret string) *AuthHandler {
	return &AuthHandler{store: store, jwtSecret: jwtSecret}
}

func generateAccessToken(userID int64, secret string, duration time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func validateToken(tokenStr string, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
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

	if len(req.Password) < 8 {
		http.Error(w, `{"error":"password must be at least 8 characters"}`, http.StatusBadRequest)
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

	accessToken, err := generateAccessToken(user.ID, h.jwtSecret, 24*time.Hour)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	refreshToken, err := generateAccessToken(user.ID, h.jwtSecret, 7*24*time.Hour)
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

	user, err := h.store.GetUserByUsername(req.Username)
	if err != nil {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	if !checkPassword(req.Password, user.PasswordHash) {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	accessToken, err := generateAccessToken(user.ID, h.jwtSecret, 24*time.Hour)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}
	refreshToken, err := generateAccessToken(user.ID, h.jwtSecret, 7*24*time.Hour)
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

	claims, err := validateToken(body.RefreshToken, h.jwtSecret)
	if err != nil {
		http.Error(w, `{"error":"invalid refresh token"}`, http.StatusUnauthorized)
		return
	}

	accessToken, err := generateAccessToken(claims.UserID, h.jwtSecret, 24*time.Hour)
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
	json.NewEncoder(w).Encode(map[string]int{"count": count})
}

// ExtractUserID extracts the authenticated user ID from the request context.
func ExtractUserID(r *http.Request) int64 {
	return apiutil.ExtractUserID(r)
}
