package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	AuthorizationHeader = "Authorization"
	BearerPrefix        = "Bearer "
	defaultTTL          = 24 * time.Hour
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrTokenExpired = errors.New("token expired")
)

type TokenManager struct {
	secret []byte
}

type Claims struct {
	UserID int64 `json:"uid"`
	jwt.RegisteredClaims
}

func NewTokenManager(secret []byte) (*TokenManager, error) {
	if len(secret) == 0 {
		return nil, errors.New("empty token secret")
	}

	secretCopy := make([]byte, len(secret))
	copy(secretCopy, secret)

	return &TokenManager{secret: secretCopy}, nil
}

func (tm *TokenManager) Generate(userID int64, now time.Time) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(defaultTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(tm.secret)
}

func (tm *TokenManager) Parse(token string, now time.Time) (int64, error) {
	var claims Claims

	parsed, err := jwt.ParseWithClaims(
		token,
		&claims,
		func(_ *jwt.Token) (any, error) {
			return tm.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithTimeFunc(func() time.Time { return now }),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return 0, ErrTokenExpired
		}
		return 0, ErrInvalidToken
	}
	if !parsed.Valid || claims.UserID <= 0 {
		return 0, ErrInvalidToken
	}
	return claims.UserID, nil
}
