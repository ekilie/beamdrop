package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ekilie/beamdrop/config"
	"github.com/ekilie/beamdrop/pkg/crypto"
	"github.com/ekilie/beamdrop/pkg/db"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidPassword = errors.New("invalid password")
	ErrNoPasswordSet   = errors.New("no password set")
	ErrInvalidToken    = errors.New("invalid token")
	jwtSecret          []byte
	// revokedTokens is an in-memory set of revoked token JTIs
	revokedTokens   = make(map[string]time.Time)
	revokedTokensMu sync.RWMutex
)

// InitJWTSecret initialises the JWT signing key. It accepts an explicit secret
// (from -jwt-secret / BEAMDROP_JWT_SECRET).
//
// Resolution order:
//  1. If `secret` is non-empty, use it (must be ≥ 32 bytes).
//  2. Else, try to load from <sharedDir>/.beamdrop/jwt_secret.
//  3. Else, generate a random 32-byte key and persist it to that file.
//
// This must be called once during startup, before any JWT operations.
func InitJWTSecret(secret string, sharedDir string) error {
	const minLen = 32
	persistDir := filepath.Join(sharedDir, config.ConfigDirName)
	if err := os.MkdirAll(persistDir, 0755); err != nil {
		return fmt.Errorf("failed to create jwt secret directory %s: %w", persistDir, err)
	}
	secretFile := filepath.Join(persistDir, "jwt_secret")

	switch {
	case secret != "":
		// Explicit secret provided via flag/env var
		if len(secret) < minLen {
			return fmt.Errorf("jwt-secret must be at least %d bytes, got %d", minLen, len(secret))
		}
		jwtSecret = []byte(secret)
		slog.Info("Using JWT secret from configuration")

	default:
		// Try to load from file
		data, err := os.ReadFile(secretFile)
		if err == nil && len(data) >= minLen {
			jwtSecret = data
			slog.Info("Loaded JWT secret from file", "path", secretFile)
		} else {
			// Generate a new random secret
			jwtSecret = make([]byte, 32)
			if _, err := rand.Read(jwtSecret); err != nil {
				return fmt.Errorf("failed to generate JWT secret: %w", err)
			}
			// Persist it (persistDir was created above)
			if err := os.WriteFile(secretFile, jwtSecret, 0600); err != nil {
				return fmt.Errorf("failed to persist JWT secret to %s: %w", secretFile, err)
			}
			slog.Info("Generated and persisted new JWT secret", "path", secretFile)
		}
	}

	// Share the key with the crypto package for at-rest encryption
	crypto.SetEncryptionKey(jwtSecret)
	return nil
}

// EncryptionKey returns the 32-byte key used for encrypting secrets at rest.
func EncryptionKey() []byte {
	return jwtSecret
}

// Claims represents JWT claims
type Claims struct {
	jwt.RegisteredClaims
}

// PasswordService handles password operations
type PasswordService struct {
	passwordHash string
	enabled      bool
}

// NewPasswordService creates a new password service
func NewPasswordService(password string) *PasswordService {
	ps := &PasswordService{
		enabled: password != "",
	}

	if password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			slog.Error("Failed to hash password", "error", err)
			return ps
		}
		ps.passwordHash = string(hash)

		// Store hash in database
		if err := ps.storePasswordHash(); err != nil {
			slog.Error("Failed to store password hash", "error", err)
		}
	}

	return ps
}

// IsEnabled returns whether password authentication is enabled
func (ps *PasswordService) IsEnabled() bool {
	return ps.enabled
}

// ValidatePassword checks if the provided password matches the stored hash
func (ps *PasswordService) ValidatePassword(password string) bool {
	if !ps.enabled {
		return true
	}

	err := bcrypt.CompareHashAndPassword([]byte(ps.passwordHash), []byte(password))
	return err == nil
}

// GenerateToken creates a new JWT token with a unique JTI for revocation support
func (ps *PasswordService) GenerateToken() (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)

	// Generate a unique JTI for revocation tracking
	jtiBytes := make([]byte, 16)
	if _, err := rand.Read(jtiBytes); err != nil {
		return "", err
	}
	jti := hex.EncodeToString(jtiBytes)

	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "beamdrop",
			ID:        jti,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken checks if the token is valid and not revoked
func (ps *PasswordService) ValidateToken(tokenString string) bool {
	if !ps.enabled {
		return true
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return false
	}

	if !token.Valid {
		return false
	}

	// Check if the token has been revoked
	if claims.ID != "" {
		revokedTokensMu.RLock()
		_, revoked := revokedTokens[claims.ID]
		revokedTokensMu.RUnlock()
		if revoked {
			return false
		}
	}

	return true
}

// RevokeToken extracts the JTI from a token and adds it to the revocation list.
func RevokeToken(tokenString string) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return jwtSecret, nil
	})
	// Even if the token is expired, we still revoke by JTI
	if err != nil && claims.ID == "" {
		return
	}
	if claims.ID == "" {
		return
	}

	expiry := time.Now().Add(24 * time.Hour) // keep until token would have expired
	if claims.ExpiresAt != nil {
		expiry = claims.ExpiresAt.Time
	}

	revokedTokensMu.Lock()
	revokedTokens[claims.ID] = expiry
	revokedTokensMu.Unlock()

	slog.Debug("Token revoked", "jti", claims.ID)
}

// CleanupRevokedTokens removes expired entries from the revocation list.
// Should be called periodically.
func CleanupRevokedTokens() {
	now := time.Now()
	revokedTokensMu.Lock()
	for jti, expiry := range revokedTokens {
		if now.After(expiry) {
			delete(revokedTokens, jti)
		}
	}
	revokedTokensMu.Unlock()
}

// StartRevocationCleanup starts a background goroutine to periodically
// clean up expired revoked tokens. Returns a stop function.
func StartRevocationCleanup() func() {
	stopCh := make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				CleanupRevokedTokens()
			}
		}
	}()
	return func() { close(stopCh) }
}

// storePasswordHash stores the password hash in the database
func (ps *PasswordService) storePasswordHash() error {
	database := db.GetDB()
	config := db.Config{
		Password: ps.passwordHash,
	}

	// Delete any existing config and create new one
	database.Where("1=1").Delete(&db.Config{})
	return database.Create(&config).Error
}

// GenerateSessionID generates a random session ID
func GenerateSessionID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}
