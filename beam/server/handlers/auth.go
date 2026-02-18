package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/ekilie/beamdrop/pkg/auth"
	"github.com/ekilie/beamdrop/pkg/metrics"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	passwordService *auth.PasswordService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(ps *auth.PasswordService) *AuthHandler {
	return &AuthHandler{
		passwordService: ps,
	}
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Password string `json:"password"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	Message string `json:"message,omitempty"`
}

// AuthStatusResponse represents the auth status response
type AuthStatusResponse struct {
	AuthEnabled   bool `json:"authEnabled"`
	Authenticated bool `json:"authenticated"`
}

// Login handles the login endpoint
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(LoginResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	if !h.passwordService.ValidatePassword(req.Password) {
		slog.Warn("Failed login attempt")
		metrics.AuthFailuresTotal.WithLabelValues("invalid_password").Inc()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(LoginResponse{
			Success: false,
			Message: "Invalid password",
		})
		return
	}

	token, err := h.passwordService.GenerateToken()
	if err != nil {
		slog.Error("Failed to generate token", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(LoginResponse{
			Success: false,
			Message: "Failed to generate authentication token",
		})
		return
	}

	// Set token as cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "beamdrop_token",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	slog.Info("Successful login")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{
		Success: true,
		Token:   token,
		Message: "Login successful",
	})
}

// Logout handles the logout endpoint
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Clear the cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "beamdrop_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Now().Add(-1 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "Logged out successfully",
	})
}

// Status handles the auth status endpoint
func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authenticated := false

	// If auth is not enabled, everyone is "authenticated"
	if !h.passwordService.IsEnabled() {
		authenticated = true
	} else {
		// Check for valid token in cookie
		cookie, err := r.Cookie("beamdrop_token")
		if err == nil && cookie.Value != "" {
			authenticated = h.passwordService.ValidateToken(cookie.Value)
		}

		// Also check Authorization header
		if !authenticated {
			authHeader := r.Header.Get("Authorization")
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token := authHeader[7:]
				authenticated = h.passwordService.ValidateToken(token)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthStatusResponse{
		AuthEnabled:   h.passwordService.IsEnabled(),
		Authenticated: authenticated,
	})
}
