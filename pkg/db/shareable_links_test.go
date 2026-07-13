package db

import (
	"testing"
	"time"

	"github.com/ekilie/beamdrop/pkg/crypto"
)

func TestGenerateToken(t *testing.T) {
	t1, err := GenerateToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(t1) != 32 { // 16 bytes = 32 hex chars
		t.Fatalf("expected token length 32, got %d", len(t1))
	}

	t2, _ := GenerateToken()
	if t1 == t2 {
		t.Fatal("tokens should be unique")
	}
}

func TestCreateShareableLink(t *testing.T) {
	setupTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	link, err := CreateShareableLink("/path/to/file.txt", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if link.Path != "/path/to/file.txt" {
		t.Fatalf("expected path '/path/to/file.txt', got %q", link.Path)
	}
	if link.Token == "" {
		t.Fatal("expected non-empty token")
	}
	if link.AccessCount != 0 {
		t.Fatalf("expected access count 0, got %d", link.AccessCount)
	}
}

func TestCreateShareableLink_WithPassword(t *testing.T) {
	setupTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	link, err := CreateShareableLink("/path", "secret123", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if link.PasswordHash == "" {
		t.Fatal("expected non-empty password hash")
	}
}

func TestCreateShareableLink_WithExpiry(t *testing.T) {
	setupTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	expiry := 1 * time.Hour
	link, err := CreateShareableLink("/path", "", &expiry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if link.ExpiresAt == nil {
		t.Fatal("expected non-nil ExpiresAt")
	}
}

func TestGetShareableLinkByToken(t *testing.T) {
	setupTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	created, _ := CreateShareableLink("/path/to/file.txt", "", nil)

	found, err := GetShareableLinkByToken(created.Token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find link")
	}
	if found.Path != "/path/to/file.txt" {
		t.Fatalf("expected path '/path/to/file.txt', got %q", found.Path)
	}
}

func TestGetShareableLinkByToken_NotFound(t *testing.T) {
	setupTestDB(t)

	found, err := GetShareableLinkByToken("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != nil {
		t.Fatal("expected nil for non-existent token")
	}
}

func TestGetShareableLinkByToken_Expired(t *testing.T) {
	setupTestDB(t)

	past := time.Now().Add(-1 * time.Hour)
	link := &ShareableLink{
		Path:      "/path",
		Token:     "expiredtoken12345678",
		ExpiresAt: &past,
		CreatedAt: time.Now(),
	}
	db.Create(link)

	_, err := GetShareableLinkByToken("expiredtoken12345678")
	if err == nil {
		t.Fatal("expected error for expired link")
	}
}

func TestValidateLinkPassword(t *testing.T) {
	setupTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	// No password
	link := &ShareableLink{PasswordHash: ""}
	if !ValidateLinkPassword(link, "") {
		t.Fatal("link without password should accept any")
	}

	// With bcrypt password
	created, _ := CreateShareableLink("/path", "mypassword", nil)
	if !ValidateLinkPassword(created, "mypassword") {
		t.Fatal("correct password should validate")
	}
	if ValidateLinkPassword(created, "wrong") {
		t.Fatal("wrong password should not validate")
	}
}

func TestIncrementAccessCount(t *testing.T) {
	setupTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	created, _ := CreateShareableLink("/path", "", nil)

	if err := IncrementAccessCount(created.Token); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found, _ := GetShareableLinkByToken(created.Token)
	if found.AccessCount != 1 {
		t.Fatalf("expected access count 1, got %d", found.AccessCount)
	}
}

func TestListShareableLinks(t *testing.T) {
	setupTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	CreateShareableLink("/path1", "", nil)
	CreateShareableLink("/path2", "", nil)

	links, err := ListShareableLinks()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}
}

func TestDeleteShareableLink(t *testing.T) {
	setupTestDB(t)
	key := make([]byte, 32)
	crypto.SetEncryptionKey(key)

	created, _ := CreateShareableLink("/path", "", nil)

	if err := DeleteShareableLink(created.Token); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found, _ := GetShareableLinkByToken(created.Token)
	if found != nil {
		t.Fatal("link should be deleted")
	}
}

func TestDeleteShareableLink_NotFound(t *testing.T) {
	setupTestDB(t)

	err := DeleteShareableLink("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent link")
	}
}

func TestCleanupExpiredLinks(t *testing.T) {
	setupTestDB(t)

	// Create an expired link
	past := time.Now().Add(-1 * time.Hour)
	db.Create(&ShareableLink{
		Path:      "/old",
		Token:     "old_token_1234567890",
		ExpiresAt: &past,
		CreatedAt: time.Now(),
	})

	// Create a valid link
	future := time.Now().Add(1 * time.Hour)
	db.Create(&ShareableLink{
		Path:      "/new",
		Token:     "new_token_1234567890",
		ExpiresAt: &future,
		CreatedAt: time.Now(),
	})

	if err := CleanupExpiredLinks(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	links, _ := ListShareableLinks()
	if len(links) != 1 {
		t.Fatalf("expected 1 link after cleanup, got %d", len(links))
	}
	if links[0].Token != "new_token_1234567890" {
		t.Fatalf("expected valid link to remain, got %q", links[0].Token)
	}
}
