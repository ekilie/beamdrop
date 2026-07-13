package crypto

import (
	"testing"
	"time"
)

func TestHashSecret(t *testing.T) {
	h1 := HashSecret("hello")
	h2 := HashSecret("hello")
	h3 := HashSecret("world")

	if h1 != h2 {
		t.Fatal("same input should produce same hash")
	}
	if h1 == h3 {
		t.Fatal("different input should produce different hash")
	}
	if len(h1) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(h1))
	}
}

func TestVerifySecret(t *testing.T) {
	if !VerifySecret("hello", HashSecret("hello")) {
		t.Fatal("should verify correctly")
	}
	if VerifySecret("hello", HashSecret("world")) {
		t.Fatal("should reject wrong secret")
	}
}

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("mypassword")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
}

func TestVerifyPassword(t *testing.T) {
	hash, err := HashPassword("mypassword")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword("mypassword", hash) {
		t.Fatal("bcrypt verify should succeed")
	}
	if VerifyPassword("wrong", hash) {
		t.Fatal("bcrypt verify should reject wrong password")
	}
}

func TestVerifyPassword_LegacySHA256(t *testing.T) {
	legacyHash := HashSecret("legacy")
	if !VerifyPassword("legacy", legacyHash) {
		t.Fatal("legacy SHA-256 verify should succeed")
	}
	if VerifyPassword("wrong", legacyHash) {
		t.Fatal("legacy SHA-256 verify should reject wrong password")
	}
}

func TestGenerateSignature(t *testing.T) {
	sig := GenerateSignature("secret", "GET", "/bucket/key", "2024-01-01T00:00:00Z")
	if sig == "" {
		t.Fatal("expected non-empty signature")
	}
}

func TestVerifySignature(t *testing.T) {
	secret := "my-secret"
	method := "PUT"
	path := "/bucket/key"
	ts := "2024-01-01T00:00:00Z"

	sig := GenerateSignature(secret, method, path, ts)
	if !VerifySignature(secret, method, path, ts, sig) {
		t.Fatal("should verify correct signature")
	}
	if VerifySignature("wrong-secret", method, path, ts, sig) {
		t.Fatal("should reject wrong secret")
	}
	if VerifySignature(secret, "GET", path, ts, sig) {
		t.Fatal("should reject wrong method")
	}
	if VerifySignature(secret, method, "/different", ts, sig) {
		t.Fatal("should reject wrong path")
	}
}

func TestGeneratePresignedToken(t *testing.T) {
	expires := time.Now().Add(1 * time.Hour)
	token := GeneratePresignedToken("secret", "GET", "bucket", "key", expires)
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestVerifyPresignedToken(t *testing.T) {
	secret := "my-secret"
	expires := time.Now().Add(1 * time.Hour)

	token := GeneratePresignedToken(secret, "GET", "bucket", "key", expires)
	if !VerifyPresignedToken(secret, "GET", "bucket", "key", expires, token) {
		t.Fatal("should verify correct token")
	}
	if VerifyPresignedToken("wrong", "GET", "bucket", "key", expires, token) {
		t.Fatal("should reject wrong secret")
	}
	if VerifyPresignedToken(secret, "PUT", "bucket", "key", expires, token) {
		t.Fatal("should reject wrong method")
	}
}

func TestIsTimestampValid(t *testing.T) {
	valid := time.Now().UTC().Format(time.RFC3339)
	if !IsTimestampValid(valid) {
		t.Fatal("current timestamp should be valid")
	}

	if IsTimestampValid("invalid-date") {
		t.Fatal("invalid date should be rejected")
	}

	if IsTimestampValid("") {
		t.Fatal("empty string should be rejected")
	}

	past := time.Now().Add(-1 * time.Minute).UTC().Format(time.RFC3339)
	if !IsTimestampValid(past) {
		t.Fatal("1 minute old timestamp should be valid (within 15 min window)")
	}

	farPast := time.Now().Add(-30 * time.Minute).UTC().Format(time.RFC3339)
	if IsTimestampValid(farPast) {
		t.Fatal("30 minute old timestamp should be rejected")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	plaintext := "hello world, this is sensitive data"
	encrypted, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	if encrypted == plaintext {
		t.Fatal("encrypted output should differ from plaintext")
	}

	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("decrypt mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncrypt_WrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	key2[0] = 1

	encrypted, err := Encrypt("secret", key1)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Decrypt(encrypted, key2)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong key")
	}
}

func TestEncrypt_InvalidKeySize(t *testing.T) {
	_, err := Encrypt("data", []byte("short"))
	if err == nil {
		t.Fatal("expected error with short key")
	}
}

func TestDecrypt_InvalidBase64(t *testing.T) {
	key := make([]byte, 32)
	_, err := Decrypt("not-base64!!!", key)
	if err == nil {
		t.Fatal("expected error with invalid base64")
	}
}

func TestDecrypt_ShortCiphertext(t *testing.T) {
	key := make([]byte, 32)
	_, err := Decrypt("AAEC", key)
	if err == nil {
		t.Fatal("expected error with short ciphertext")
	}
}

func TestEncrypt_DifferentNonces(t *testing.T) {
	key := make([]byte, 32)
	e1, _ := Encrypt("same data", key)
	e2, _ := Encrypt("same data", key)
	if e1 == e2 {
		t.Fatal("encrypted outputs should differ due to random nonce")
	}
}

func TestSetEncryptionKey(t *testing.T) {
	key := []byte("01234567890123456789012345678901")
	SetEncryptionKey(key)
	got := GetEncryptionKey()
	if len(got) != len(key) {
		t.Fatalf("expected key length %d, got %d", len(key), len(got))
	}
	for i := range key {
		if got[i] != key[i] {
			t.Fatalf("key mismatch at byte %d", i)
		}
	}

	// Verify it's a copy, not a reference
	key[0] = 99
	if got[0] == 99 {
		t.Fatal("GetEncryptionKey should return a copy")
	}
}

func TestEncryptDecrypt_WithKeyStore(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	SetEncryptionKey(key)

	encrypted, err := Encrypt("sensitive", GetEncryptionKey())
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := Decrypt(encrypted, GetEncryptionKey())
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != "sensitive" {
		t.Fatalf("expected 'sensitive', got %q", decrypted)
	}
}
