package platform

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	jwt.RegisteredClaims
	UPN         string   `json:"upn,omitempty"`
	Groups      []string `json:"groups"`
	Permissions []string `json:"permissions"`
}

func SignHS256(secret, keyID, issuer, subject, upn string, groups, permissions []string, lifespan time.Duration) (string, error) {
	if groups == nil {
		groups = []string{}
	}
	if permissions == nil {
		permissions = []string{}
	}
	now := time.Now().UTC()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   subject,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(lifespan)),
		},
		UPN:         upn,
		Groups:      groups,
		Permissions: permissions,
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tok.Header["kid"] = keyID
	return tok.SignedString([]byte(secret))
}

func ParseHS256(secret, issuer, raw string) (*Claims, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
	)
	claims := &Claims{}
	tok, err := parser.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil || !tok.Valid {
		if err == nil {
			err = fmt.Errorf("invalid token")
		}
		return nil, err
	}
	return claims, nil
}
