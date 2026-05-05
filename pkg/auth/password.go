package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log/slog"
	"time"

	"github.com/ekilie/beamdrop/pkg/db"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidPassword = errors.New("invalid password")
	ErrNoPasswordSet   = errors.New("no password set")
	ErrInvalidToken    = errors.New("invalid token")
	jwtSecret          []byte
)

func init() {
	// Generate a random JWT secret on startup
	jwtSecret = make([]byte, 32)
	if _, err := rand.Read(jwtSecret); err != nil {
		panic("CRITICAL: failed to generate JWT secret: " + err.Error())
	}
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

// GenerateToken creates a new JWT token
func (ps *PasswordService) GenerateToken() (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "beamdrop",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken checks if the token is valid
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

	return token.Valid
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
