package internal

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	tokenSubject = "access"
	tokenIssuer  = "my-service"        // поменяй на имя своего сервиса
	tokenTTL     = 90 * 24 * time.Hour // ~3 месяца (90 дней)
)

type Claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

func MakeToken(userID int64, secret string) (string, error) {
	now := time.Now()

	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   tokenSubject,
			Issuer:    tokenIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

func ParseToken(tokenStr string, secret string) (int64, error) {
	t, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (any, error) {
		// 1) Жёстко проверяем алгоритм
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return 0, errors.New("invalid token")
	}

	claims, ok := t.Claims.(*Claims)
	if !ok || !t.Valid || claims.UserID <= 0 {
		return 0, errors.New("invalid token")
	}

	// 2) Проверяем, что это именно access token
	if claims.Subject != tokenSubject {
		return 0, errors.New("invalid token")
	}

	// 3) Проверяем, что токен выдан нашим сервисом
	if claims.Issuer != tokenIssuer {
		return 0, errors.New("invalid token")
	}

	return claims.UserID, nil
}
