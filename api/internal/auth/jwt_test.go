package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestTokenServiceIssueAndValidate(t *testing.T) {
	svc := NewTokenService("test-secret-key-32bytes!!", 72)
	userID := uuid.New()
	email := "test@example.com"

	tokenStr, expiry, err := svc.Issue(userID, email)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("token string is empty")
	}
	if expiry.Before(time.Now()) {
		t.Error("expiry is in the past")
	}

	claims, err := svc.Validate(tokenStr)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("UserID = %v, want %v", claims.UserID, userID)
	}
	if claims.Email != email {
		t.Errorf("Email = %q, want %q", claims.Email, email)
	}
}

func TestTokenServiceValidate(t *testing.T) {
	svc := NewTokenService("test-secret-key-32bytes!!", 72)

	// Create a valid token for some test cases
	userID := uuid.New()
	validToken, _, err := svc.Issue(userID, "test@example.com")
	if err != nil {
		t.Fatalf("creating test token: %v", err)
	}

	tests := []struct {
		name    string
		token   string
		wantErr error
	}{
		{
			name:  "valid token",
			token: validToken,
		},
		{
			name:    "empty string",
			token:   "",
			wantErr: ErrInvalidToken,
		},
		{
			name:    "malformed string",
			token:   "not.a.jwt",
			wantErr: ErrInvalidToken,
		},
		{
			name: "wrong signing key",
			token: func() string {
				other := NewTokenService("different-secret-key!!!!", 72)
				tok, _, err := other.Issue(uuid.New(), "x@x.com")
				if err != nil {
					t.Fatalf("issuing token with wrong key: %v", err)
				}
				return tok
			}(),
			wantErr: ErrInvalidToken,
		},
		{
			name: "expired token",
			token: func() string {
				claims := Claims{
					RegisteredClaims: jwt.RegisteredClaims{
						Subject:   userID.String(),
						IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
					},
					UserID: userID,
					Email:  "test@example.com",
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				signed, err := token.SignedString(svc.secret)
				if err != nil {
					t.Fatalf("signing expired token: %v", err)
				}
				return signed
			}(),
			wantErr: ErrExpiredToken,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			claims, err := svc.Validate(tc.token)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got error %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if claims.UserID != userID {
				t.Errorf("UserID = %v, want %v", claims.UserID, userID)
			}
		})
	}
}
