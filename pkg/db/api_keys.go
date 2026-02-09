package db

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/tachRoutine/beamdrop-go/pkg/logger"
	"gorm.io/gorm"
)

// APIKey represents an API key for S3-like API access
type APIKey struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Name        string     `gorm:"size:255;not null" json:"name"`
	AccessKeyID string     `gorm:"column:access_key_id;uniqueIndex;size:24;not null" json:"accessKeyId"`
	SecretKey   string     `gorm:"column:secret_key;size:64;not null" json:"-"` // Stored for HMAC verification
	Permissions string     `gorm:"type:text" json:"permissions"`                // JSON permissions
	BucketScope string     `gorm:"column:bucket_scope;size:255" json:"bucketScope,omitempty"`
	ExpiresAt   *time.Time `gorm:"column:expires_at" json:"expiresAt,omitempty"`
	LastUsedAt  *time.Time `gorm:"column:last_used_at" json:"lastUsedAt,omitempty"`
	CreatedAt   time.Time  `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"createdAt"`
	Disabled    bool       `gorm:"default:false" json:"disabled"`
}

func (APIKey) TableName() string {
	return "api_keys"
}

// GenerateKeyPair creates a new access key ID and secret key
func GenerateKeyPair() (accessKeyID, secretKey string, err error) {
	// Generate access key ID (BDK_ + 16 hex chars)
	accessBytes := make([]byte, 8)
	if _, err := rand.Read(accessBytes); err != nil {
		return "", "", err
	}
	accessKeyID = "BDK_" + hex.EncodeToString(accessBytes)

	// Generate secret key (sk_ + 40 hex chars)
	secretBytes := make([]byte, 20)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", "", err
	}
	secretKey = "sk_" + hex.EncodeToString(secretBytes)

	return accessKeyID, secretKey, nil
}

// CreateAPIKey creates a new API key and returns the secret (only shown once)
func CreateAPIKey(name string, permissions string, bucketScope string, expiresIn *time.Duration) (*APIKey, string, error) {
	accessKeyID, secretKey, err := GenerateKeyPair()
	if err != nil {
		logger.Error("Failed to generate key pair: %v", err)
		return nil, "", err
	}

	var expiresAt *time.Time
	if expiresIn != nil {
		t := time.Now().Add(*expiresIn)
		expiresAt = &t
	}

	apiKey := &APIKey{
		Name:        name,
		AccessKeyID: accessKeyID,
		SecretKey:   secretKey, // Store secret for HMAC verification
		Permissions: permissions,
		BucketScope: bucketScope,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
	}

	db := GetDB()
	if err := db.Create(apiKey).Error; err != nil {
		logger.Error("Failed to create API key: %v", err)
		return nil, "", err
	}

	logger.Info("API key created: %s (%s)", name, accessKeyID)
	return apiKey, secretKey, nil
}

// GetAPIKeyByAccessID retrieves an API key by its access key ID
func GetAPIKeyByAccessID(accessKeyID string) (*APIKey, error) {
	db := GetDB()
	var apiKey APIKey
	err := db.Where("access_key_id = ? AND disabled = ?", accessKeyID, false).First(&apiKey).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	// Check expiration
	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return nil, nil
	}

	return &apiKey, nil
}

// UpdateLastUsed updates the last used timestamp for an API key
func UpdateLastUsed(accessKeyID string) error {
	db := GetDB()
	now := time.Now()
	return db.Model(&APIKey{}).Where("access_key_id = ?", accessKeyID).Update("last_used_at", now).Error
}

// ListAPIKeys returns all API keys (without secrets)
func ListAPIKeys() ([]APIKey, error) {
	db := GetDB()
	var keys []APIKey
	err := db.Order("created_at DESC").Find(&keys).Error
	if err != nil {
		logger.Error("Failed to list API keys: %v", err)
		return nil, err
	}
	return keys, nil
}

// DeleteAPIKey deletes an API key by access key ID
func DeleteAPIKey(accessKeyID string) error {
	db := GetDB()
	result := db.Where("access_key_id = ?", accessKeyID).Delete(&APIKey{})
	if result.Error != nil {
		logger.Error("Failed to delete API key: %v", result.Error)
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("API key not found")
	}
	logger.Info("API key deleted: %s", accessKeyID)
	return nil
}

// DisableAPIKey disables an API key without deleting it
func DisableAPIKey(accessKeyID string) error {
	db := GetDB()
	result := db.Model(&APIKey{}).Where("access_key_id = ?", accessKeyID).Update("disabled", true)
	if result.Error != nil {
		logger.Error("Failed to disable API key: %v", result.Error)
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("API key not found")
	}
	logger.Info("API key disabled: %s", accessKeyID)
	return nil
}
