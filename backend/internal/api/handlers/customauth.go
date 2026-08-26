package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	DB        *sql.DB
	JWTSecret string
	AppURL    string
}

type customClaims struct {
	Username string `json:"username"`
	IsAdmin  bool   `json:"is_admin"`
	jwt.RegisteredClaims
}

func (h *AuthHandler) mintToken(userID, username string, isAdmin bool) (string, error) {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, customClaims{
		Username: username,
		IsAdmin:  isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
		},
	})
	return tok.SignedString([]byte(h.JWTSecret))
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || len(req.Password) < 6 {
		writeJSONError(w, http.StatusBadRequest, "username required and password min 6 chars")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	var userID string
	err = h.DB.QueryRowContext(r.Context(),
		`INSERT INTO profiles (username, password_hash) VALUES ($1, $2) RETURNING id`,
		req.Username, string(hash),
	).Scan(&userID)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			writeJSONError(w, http.StatusConflict, "username already taken")
			return
		}
		log.Printf("register: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "could not create account")
		return
	}
	tok, err := h.mintToken(userID, req.Username, false)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "session error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": tok, "id": userID, "username": req.Username})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	var userID, hash string
	var isAdmin bool
	err := h.DB.QueryRowContext(r.Context(),
		`SELECT id, COALESCE(password_hash,''), is_admin FROM profiles
		 WHERE username = $1 AND is_blocked = false`, req.Username,
	).Scan(&userID, &hash, &isAdmin)
	if err != nil || hash == "" {
		writeJSONError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}
	if err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		writeJSONError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}
	tok, err := h.mintToken(userID, req.Username, isAdmin)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "session error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": tok, "id": userID, "username": req.Username})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	// временная заглушка
	writeJSONError(w, http.StatusNotImplemented, "not implemented")
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	writeJSONError(w, http.StatusNotImplemented, "not implemented")
}