package auth

import (
	"testing"
)

func TestPasswordService_Disabled(t *testing.T) {
	ps := NewPasswordService("")
	if ps.IsEnabled() {
		t.Fatal("empty password should disable auth")
	}
	if !ps.ValidatePassword("anything") {
		t.Fatal("disabled auth should accept any password")
	}
	token, err := ps.GenerateToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ps.ValidateToken(token) {
		t.Fatal("disabled auth should validate any token")
	}
}

func TestPasswordService_ValidatePassword(t *testing.T) {
	ps := NewPasswordService("correct-password")
	if !ps.IsEnabled() {
		t.Fatal("non-empty password should enable auth")
	}
	if !ps.ValidatePassword("correct-password") {
		t.Fatal("correct password should validate")
	}
	if ps.ValidatePassword("wrong-password") {
		t.Fatal("wrong password should not validate")
	}
}

func TestPasswordService_GenerateAndValidateToken(t *testing.T) {
	ps := NewPasswordService("test-password")
	token, err := ps.GenerateToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if !ps.ValidateToken(token) {
		t.Fatal("valid token should validate")
	}
}

func TestPasswordService_InvalidateToken(t *testing.T) {
	ps := NewPasswordService("test-password")
	token, err := ps.GenerateToken()
	if err != nil {
		t.Fatal(err)
	}

	RevokeToken(token)
	if ps.ValidateToken(token) {
		t.Fatal("revoked token should not validate")
	}
}

func TestRevokeToken_InvalidToken(t *testing.T) {
	RevokeToken("invalid-token")
}

func TestRevokeToken_EmptyJTI(t *testing.T) {
	RevokeToken("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c")
}

func TestCleanupRevokedTokens(t *testing.T) {
	ps := NewPasswordService("pass")
	token, err := ps.GenerateToken()
	if err != nil {
		t.Fatal(err)
	}
	RevokeToken(token)
	CleanupRevokedTokens()
}

func TestStartRevocationCleanup(t *testing.T) {
	stop := StartRevocationCleanup()
	stop()
}

func TestGenerateSessionID(t *testing.T) {
	id1 := GenerateSessionID()
	id2 := GenerateSessionID()
	if id1 == "" {
		t.Fatal("expected non-empty session ID")
	}
	if id1 == id2 {
		t.Fatal("session IDs should be unique")
	}
}
