package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
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

// Encrypt encrypts plaintext using AES-256-GCM with the given 32-byte key.
// Returns a base64-encoded ciphertext (nonce prepended).
func Encrypt(plaintext string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes.NewCipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("cipher.NewGCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce generation: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64-encoded AES-256-GCM ciphertext with the given 32-byte key.
func Decrypt(encoded string, key []byte) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes.NewCipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("cipher.NewGCM: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("gcm.Open: %w", err)
	}
	return string(plaintext), nil
}
