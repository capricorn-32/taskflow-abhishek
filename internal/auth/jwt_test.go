package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWTGenerateAndParseRoundTrip(t *testing.T) {
	m := NewJWTManager("secret-key", "taskflow", 24*time.Hour)
	uid := uuid.New()

	token, err := m.GenerateToken(uid, "user@example.com")
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	claims, err := m.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken returned error: %v", err)
	}

	if claims.UserID != uid.String() {
		t.Fatalf("unexpected user_id claim: got %q want %q", claims.UserID, uid.String())
	}
	if claims.Email != "user@example.com" {
		t.Fatalf("unexpected email claim: got %q", claims.Email)
	}
	if claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(time.Now()) {
		t.Fatalf("token expiry claim missing or already expired")
	}
}

func TestJWTRejectsInvalidToken(t *testing.T) {
	m := NewJWTManager("secret-key", "taskflow", 24*time.Hour)

	if _, err := m.ParseToken("not-a-jwt"); err == nil {
		t.Fatal("expected ParseToken to fail for invalid token")
	}
}
