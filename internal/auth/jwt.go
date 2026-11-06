package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	AccountID uint   `json:"account_id"`
	Username  string `json:"username"`
	jwt.RegisteredClaims
}

func jwtSecret() ([]byte, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	if len(secret) < 32 {
		return nil, errors.New("JWT_SECRET must contain at least 32 bytes")
	}
	return secret, nil
}

func GenerateToken(accountID uint, username string) (string, error) {
	secret, err := jwtSecret()
	if err != nil {
		return "", err
	}
	id, err := GenerateRefreshToken(accountID)
	if err != nil {
		return "", err
	}
	now := time.Now()
	claims := Claims{AccountID: accountID, Username: username, RegisteredClaims: jwt.RegisteredClaims{
		ID:        id,
		ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
	}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

func GenerateRefreshToken(accountID uint) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func ParseToken(tokenString string) (*Claims, error) {
	secret, err := jwtSecret()
	if err != nil {
		return nil, err
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) { return secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return nil, err
	}
	if !token.Valid || claims.AccountID == 0 {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}
