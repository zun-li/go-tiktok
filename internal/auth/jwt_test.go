package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestTokenValidation(t *testing.T) {
	const secret = "test-signing-secret-at-least-32-bytes"
	t.Setenv("JWT_SECRET", secret)
	first, err := GenerateToken(42, "alice")
	if err != nil {
		t.Fatal(err)
	}
	second, err := GenerateToken(42, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("separate sessions must have different access tokens")
	}
	parsed, err := ParseToken(first)
	if err != nil || parsed.AccountID != 42 || parsed.Username != "alice" {
		t.Fatalf("round trip: claims=%v error=%v", parsed, err)
	}
	cases := []struct {
		name   string
		method jwt.SigningMethod
		claims Claims
		key    any
	}{
		{"expired", jwt.SigningMethodHS256, Claims{AccountID: 42, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour))}}, []byte(secret)},
		{"missing expiry", jwt.SigningMethodHS256, Claims{AccountID: 42}, []byte(secret)},
		{"zero account", jwt.SigningMethodHS256, Claims{RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}, []byte(secret)},
		{"wrong signature", jwt.SigningMethodHS256, Claims{AccountID: 42, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}, []byte("wrong-secret")},
		{"wrong algorithm", jwt.SigningMethodHS384, Claims{AccountID: 42}, []byte(secret)},
		{"unsigned", jwt.SigningMethodNone, Claims{AccountID: 42}, jwt.UnsafeAllowNoneSignatureType},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := jwt.NewWithClaims(tc.method, tc.claims).SignedString(tc.key)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ParseToken(raw); err == nil {
				t.Fatal("invalid token accepted")
			}
		})
	}
	t.Setenv("JWT_SECRET", "short")
	if _, err := GenerateToken(42, "alice"); err == nil {
		t.Fatal("weak key accepted")
	}
}
