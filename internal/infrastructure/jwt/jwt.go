// Package jwt verifies bearer tokens for the content service.
package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims are the HS256 claims issued by the auth service.
type Claims struct {
	jwt.RegisteredClaims
	UPN         string   `json:"upn,omitempty"`
	Groups      []string `json:"groups"`
	Permissions []string `json:"permissions"`
}

// SignHS256 signs a token with the shared HMAC secret.
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
	var tok *jwt.Token
	tok = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tok.Header["kid"] = keyID
	return tok.SignedString([]byte(secret))
}

// ParseHS256 validates a bearer token against the issuer and secret.
func ParseHS256(secret, issuer, raw string) (*Claims, error) {
	var parser *jwt.Parser
	parser = jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
	)
	claims := &Claims{}
	var tok *jwt.Token
	var err error
	tok, err = parser.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
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
