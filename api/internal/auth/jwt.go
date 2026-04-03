package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("auth: invalid token")
	ErrExpiredToken = errors.New("auth: token expired")
)

// Claims represents the custom JWT claims for DoIt.
type Claims struct {
	jwt.RegisteredClaims
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
}

// TokenService handles JWT creation and validation.
type TokenService struct {
	secret      []byte
	expiryHours int
}

func NewTokenService(secret string, expiryHours int) *TokenService {
	return &TokenService{
		secret:      []byte(secret),
		expiryHours: expiryHours,
	}
}

// Issue creates a signed JWT for the given user.
func (ts *TokenService) Issue(userID uuid.UUID, email string) (string, time.Time, error) {
	now := time.Now().UTC()
	expiry := now.Add(time.Duration(ts.expiryHours) * time.Hour)

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiry),
		},
		UserID: userID,
		Email:  email,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(ts.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("signing token: %w", err)
	}

	return signed, expiry, nil
}

// Validate parses and validates a JWT string, returning the claims.
func (ts *TokenService) Validate(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return ts.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
