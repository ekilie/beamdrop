package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"
)

// HashSecret creates a SHA-256 hash of a secret key for storage
func HashSecret(secret string) string {
	h := sha256.New()
	h.Write([]byte(secret))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifySecret checks if a secret matches a stored hash
func VerifySecret(secret, hash string) bool {
	return HashSecret(secret) == hash
}

// GenerateSignature creates an HMAC-SHA256 signature for a request
func GenerateSignature(secretKey, method, path, timestamp string) string {
	message := fmt.Sprintf("%s\n%s\n%s", method, path, timestamp)
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// VerifySignature verifies an HMAC-SHA256 signature
func VerifySignature(secretKey, method, path, timestamp, signature string) bool {
	expected := GenerateSignature(secretKey, method, path, timestamp)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// GeneratePresignedToken creates a token for presigned URLs
func GeneratePresignedToken(secretKey, method, bucket, key string, expiresAt time.Time) string {
	message := fmt.Sprintf("%s\n%s\n%s\n%d", method, bucket, key, expiresAt.Unix())
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(message))
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

// VerifyPresignedToken verifies a presigned URL token
func VerifyPresignedToken(secretKey, method, bucket, key string, expiresAt time.Time, token string) bool {
	expected := GeneratePresignedToken(secretKey, method, bucket, key, expiresAt)
	return hmac.Equal([]byte(expected), []byte(token))
}

// IsTimestampValid checks if a timestamp is within acceptable range (15 minutes)
func IsTimestampValid(timestamp string) bool {
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return false
	}

	now := time.Now().UTC()
	diff := now.Sub(t)

	// Allow 15 minutes of clock skew in either direction
	return diff >= -15*time.Minute && diff <= 15*time.Minute
}
