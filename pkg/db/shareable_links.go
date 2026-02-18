package db

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"time"

	"github.com/ekilie/beamdrop/pkg/crypto"
	"gorm.io/gorm"
)

// ShareableLink represents a shareable link for a file or folder
type ShareableLink struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Path         string     `gorm:"column:path;size:1024;not null" json:"path"`             // File or folder path
	Token        string     `gorm:"column:token;uniqueIndex;size:32;not null" json:"token"` // Unique token for the link
	PasswordHash string     `gorm:"column:password_hash;size:64" json:"-"`                  // Optional password hash
	ExpiresAt    *time.Time `gorm:"column:expires_at" json:"expiresAt,omitempty"`           // Optional expiry time
	AccessCount  int        `gorm:"column:access_count;default:0" json:"accessCount"`       // Number of times accessed
	CreatedAt    time.Time  `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"createdAt"`
	CreatedBy    string     `gorm:"column:created_by;size:255" json:"createdBy,omitempty"` // Optional user identifier
}

func (ShareableLink) TableName() string {
	return "shareable_links"
}

// GenerateToken creates a unique random token for shareable links
func GenerateToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateShareableLink creates a new shareable link for a file or folder
func CreateShareableLink(path string, password string, expiresIn *time.Duration) (*ShareableLink, error) {
	token, err := GenerateToken()
	if err != nil {
		slog.Error("Failed to generate token", "error", err)
		return nil, err
	}

	var passwordHash string
	if password != "" {
		passwordHash = crypto.HashSecret(password)
	}

	var expiresAt *time.Time
	if expiresIn != nil {
		t := time.Now().Add(*expiresIn)
		expiresAt = &t
	}

	link := &ShareableLink{
		Path:         path,
		Token:        token,
		PasswordHash: passwordHash,
		ExpiresAt:    expiresAt,
		AccessCount:  0,
		CreatedAt:    time.Now(),
	}

	db := GetDB()
	if err := db.Create(link).Error; err != nil {
		slog.Error("Failed to create shareable link", "error", err)
		return nil, err
	}

	slog.Info("Shareable link created", "path", path, "token", token)
	return link, nil
}

// GetShareableLinkByToken retrieves a shareable link by its token
func GetShareableLinkByToken(token string) (*ShareableLink, error) {
	db := GetDB()
	var link ShareableLink
	err := db.Where("token = ?", token).First(&link).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	// Check expiration
	if link.ExpiresAt != nil && time.Now().After(*link.ExpiresAt) {
		return nil, errors.New("link has expired")
	}

	return &link, nil
}

// ValidateLinkPassword verifies a password for a password-protected link
func ValidateLinkPassword(link *ShareableLink, password string) bool {
	if link.PasswordHash == "" {
		return true // No password set
	}
	return crypto.VerifySecret(password, link.PasswordHash)
}

// IncrementAccessCount increments the access count for a shareable link
func IncrementAccessCount(token string) error {
	db := GetDB()
	return db.Model(&ShareableLink{}).Where("token = ?", token).
		UpdateColumn("access_count", gorm.Expr("access_count + ?", 1)).Error
}

// ListShareableLinks returns all shareable links
func ListShareableLinks() ([]ShareableLink, error) {
	db := GetDB()
	var links []ShareableLink
	err := db.Order("created_at DESC").Find(&links).Error
	if err != nil {
		slog.Error("Failed to list shareable links", "error", err)
		return nil, err
	}
	return links, nil
}

// DeleteShareableLink deletes a shareable link by token
func DeleteShareableLink(token string) error {
	db := GetDB()
	result := db.Where("token = ?", token).Delete(&ShareableLink{})
	if result.Error != nil {
		slog.Error("Failed to delete shareable link", "error", result.Error)
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("shareable link not found")
	}
	slog.Info("Shareable link deleted", "token", token)
	return nil
}

// CleanupExpiredLinks removes all expired shareable links
func CleanupExpiredLinks() error {
	db := GetDB()
	now := time.Now()
	result := db.Where("expires_at IS NOT NULL AND expires_at < ?", now).Delete(&ShareableLink{})
	if result.Error != nil {
		slog.Error("Failed to cleanup expired links", "error", result.Error)
		return result.Error
	}
	if result.RowsAffected > 0 {
		slog.Info("Cleaned up expired shareable links", "count", result.RowsAffected)
	}
	return nil
}
