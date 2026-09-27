package handler

import (
	"context"
	"net/http"

	"react-go-cms-content-service/internal/infrastructure/httpx"
	"react-go-cms-content-service/internal/infrastructure/jwt"
)

type ctxKey int

const subjectKey ctxKey = 1

func subject(r *http.Request) string {
	value, _ := r.Context().Value(subjectKey).(string)
	return value
}

func (h *Handler) public(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			h.auth(next)(w, r)
			return
		}
		next(w, r)
	}
}

func (h *Handler) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := httpx.BearerToken(r)
		if raw == "" {
			httpx.WriteError(w, httpx.Unauthorized("Unauthorized"), errorField)
			return
		}
		var claims *jwt.Claims
		var err error
		claims, err = jwt.ParseHS256(h.secret, h.issuer, raw)
		if err != nil || claims.Subject == "" {
			httpx.WriteError(w, httpx.Unauthorized("Unauthorized"), errorField)
			return
		}
		ctx := context.WithValue(r.Context(), subjectKey, claims.Subject)
		next(w, r.WithContext(ctx))
	}
}
