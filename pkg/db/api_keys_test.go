package db

import (
	"testing"
	"time"

	"github.com/ekilie/beamdrop/pkg/crypto"
)

func TestGenerateKeyPair(t *testing.T) {
	akID, secret, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(akID) != 20 { // "BDK_" + 16 hex chars
		t.Fatalf("expected access key ID length 20, got %d: %q", len(akID), akID)
	}
	if len(secret) != 43 { // "sk_" + 40 hex chars
		t.Fatalf("expected secret length 43, got %d: %q", len(secret), secret)
	}
	if akID[:4] != "BDK_" {
		t.Fatalf("expected BDK_ prefix, got %q", akID[:4])
	}
	if secret[:3] != "sk_" {
		t.Fatalf("expected sk_ prefix, got %q", secret[:3])
	}
}

func TestGenerateKeyPair_Uniqueness(t *testing.T) {
	id1, _, _ := GenerateKeyPair()
	id2, _, _ := GenerateKeyPair()
	if id1 == id2 {
		t.Fatal("key pairs should be unique")
	}
}

func TestCreateAndGetAPIKey(t *testing.T) {
	setupTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	created, secret, err := CreateAPIKey("test-key", "read", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.AccessKeyID == "" {
		t.Fatal("expected non-empty access key ID")
	}
	if secret == "" {
		t.Fatal("expected non-empty secret")
	}

	found, err := GetAPIKeyByAccessID(created.AccessKeyID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find API key")
	}
	if found.Name != "test-key" {
		t.Fatalf("expected name 'test-key', got %q", found.Name)
	}
	if found.Permissions != "read" {
		t.Fatalf("expected permissions 'read', got %q", found.Permissions)
	}
}

func TestGetAPIKey_NotFound(t *testing.T) {
	setupTestDB(t)

	found, err := GetAPIKeyByAccessID("BDK_nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != nil {
		t.Fatal("expected nil for non-existent key")
	}
}

func TestGetAPIKey_Expired(t *testing.T) {
	setupTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	past := time.Now().Add(-1 * time.Hour)
	apiKey := &APIKey{
		Name:        "expired",
		AccessKeyID: "BDK_expired123",
		SecretKey:   "encrypted",
		ExpiresAt:   &past,
		CreatedAt:   time.Now(),
	}
	db.Create(apiKey)

	found, err := GetAPIKeyByAccessID("BDK_expired123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != nil {
		t.Fatal("expected nil for expired key")
	}
}

func TestGetAPIKey_Disabled(t *testing.T) {
	setupTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	apiKey := &APIKey{
		Name:        "disabled",
		AccessKeyID: "BDK_disabled1",
		SecretKey:   "encrypted",
		Disabled:    true,
		CreatedAt:   time.Now(),
	}
	db.Create(apiKey)

	found, err := GetAPIKeyByAccessID("BDK_disabled1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != nil {
		t.Fatal("expected nil for disabled key")
	}
}

func TestUpdateLastUsed(t *testing.T) {
	setupTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	created, _, _ := CreateAPIKey("test", "read", "", nil)

	if err := UpdateLastUsed(created.AccessKeyID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found, _ := GetAPIKeyByAccessID(created.AccessKeyID)
	if found.LastUsedAt == nil {
		t.Fatal("expected LastUsedAt to be set")
	}
}

func TestListAPIKeys(t *testing.T) {
	setupTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	CreateAPIKey("key1", "read", "", nil)
	CreateAPIKey("key2", "write", "my-bucket", nil)

	keys, err := ListAPIKeys()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
}

func TestDeleteAPIKey(t *testing.T) {
	setupTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	created, _, _ := CreateAPIKey("test", "read", "", nil)

	if err := DeleteAPIKey(created.AccessKeyID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found, _ := GetAPIKeyByAccessID(created.AccessKeyID)
	if found != nil {
		t.Fatal("key should be deleted")
	}
}

func TestDeleteAPIKey_NotFound(t *testing.T) {
	setupTestDB(t)

	err := DeleteAPIKey("BDK_nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent key")
	}
}

func TestDisableAPIKey(t *testing.T) {
	setupTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	created, _, _ := CreateAPIKey("test", "read", "", nil)

	if err := DisableAPIKey(created.AccessKeyID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found, _ := GetAPIKeyByAccessID(created.AccessKeyID)
	if found != nil {
		t.Fatal("disabled key should return nil from GetAPIKeyByAccessID")
	}
}

func TestDisableAPIKey_NotFound(t *testing.T) {
	setupTestDB(t)

	err := DisableAPIKey("BDK_nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent key")
	}
}

func TestDecryptSecretKey(t *testing.T) {
	setupTestDB(t)
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	crypto.SetEncryptionKey(key)

	_, secret, _ := CreateAPIKey("test", "read", "", nil)

	created, _ := GetAPIKeyByAccessID("BDK_test123")
	if created == nil {
		// Get the key via ListAPIKeys instead
		keys, _ := ListAPIKeys()
		if len(keys) == 0 {
			t.Fatal("expected at least one key")
		}
		created = &keys[0]
	}

	decrypted, err := DecryptSecretKey(created)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decrypted != secret {
		t.Fatalf("decrypted secret doesn't match: got %q, want %q", decrypted, secret)
	}
}
