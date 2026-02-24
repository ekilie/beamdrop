package db

import (
    "crypto/rand"
    "encoding/hex"
    "errors"
    "log/slog"
    "time"

    "gorm.io/gorm"
)

type PresignedURL struct {
    ID            uint       `gorm:"primaryKey" json:"id"`
    Token         string     `gorm:"column:token;uniqueIndex;size:32;not null" json:"token"`
    Bucket        string     `gorm:"column:bucket;size:255;not null" json:"bucket"`
    Key           string     `gorm:"column:key;size:1024;not null" json:"key"`
    Method        string     `gorm:"column:method;size:10;not null;default:GET" json:"method"` // GET or PUT
    ExpiresAt     *time.Time `gorm:"column:expires_at" json:"expiresAt,omitempty"`
    MaxDownloads  *int       `gorm:"column:max_downloads" json:"maxDownloads,omitempty"`
    DownloadCount int        `gorm:"column:download_count;default:0" json:"downloadCount"`
    CreatedBy     string     `gorm:"column:created_by;size:255" json:"createdBy,omitempty"` // access_key_id
    CreatedAt     time.Time  `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"createdAt"`
}

func (PresignedURL) TableName() string {
    return "presigned_urls"
}

// generatePresignToken creates a 16-byte hex token (same as GenerateToken in shareable_links.go)
func generatePresignToken() (string, error) {
    bytes := make([]byte, 16)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return hex.EncodeToString(bytes), nil
}

func CreatePresignedURL(bucket, key, method, createdBy string, expiresIn *time.Duration, maxDownloads *int) (*PresignedURL, error) {
    token, err := generatePresignToken()
    if err != nil {
        return nil, err
    }

    var expiresAt *time.Time
    if expiresIn != nil {
        t := time.Now().Add(*expiresIn)
        expiresAt = &t
    }

    p := &PresignedURL{
        Token:        token,
        Bucket:       bucket,
        Key:          key,
        Method:       method,
        ExpiresAt:    expiresAt,
        MaxDownloads: maxDownloads,
        CreatedBy:    createdBy,
        CreatedAt:    time.Now(),
    }

    db := GetDB()
    if err := db.Create(p).Error; err != nil {
        slog.Error("Failed to create presigned URL", "error", err)
        return nil, err
    }
    slog.Info("Presigned URL created", "token", token, "bucket", bucket, "key", key)
    return p, nil
}

func GetPresignedURLByToken(token string) (*PresignedURL, error) {
    db := GetDB()
    var p PresignedURL
    err := db.Where("token = ?", token).First(&p).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil
        }
        return nil, err
    }

    // Check expiration
    if p.ExpiresAt != nil && time.Now().After(*p.ExpiresAt) {
        return nil, errors.New("presigned URL has expired")
    }

    // Check max downloads
    if p.MaxDownloads != nil && p.DownloadCount >= *p.MaxDownloads {
        return nil, errors.New("download limit reached")
    }

    return &p, nil
}

func IncrementPresignedURLDownloads(token string) error {
    db := GetDB()
    return db.Model(&PresignedURL{}).Where("token = ?", token).
        UpdateColumn("download_count", gorm.Expr("download_count + ?", 1)).Error
}

func ListPresignedURLs() ([]PresignedURL, error) {
    db := GetDB()
    var urls []PresignedURL
    err := db.Order("created_at DESC").Find(&urls).Error
    if err != nil {
        slog.Error("Failed to list presigned URLs", "error", err)
        return nil, err
    }
    return urls, nil
}

func DeletePresignedURL(token string) error {
    db := GetDB()
    result := db.Where("token = ?", token).Delete(&PresignedURL{})
    if result.Error != nil {
        return result.Error
    }
    if result.RowsAffected == 0 {
        return errors.New("presigned URL not found")
    }
    slog.Info("Presigned URL deleted", "token", token)
    return nil
}

func CleanupExpiredPresignedURLs() error {
    db := GetDB()
    now := time.Now()
    result := db.Where("expires_at IS NOT NULL AND expires_at < ?", now).Delete(&PresignedURL{})
    if result.Error != nil {
        slog.Error("Failed to cleanup expired presigned URLs", "error", result.Error)
        return result.Error
    }
    if result.RowsAffected > 0 {
        slog.Info("Cleaned up expired presigned URLs", "count", result.RowsAffected)
    }
    return nil
}