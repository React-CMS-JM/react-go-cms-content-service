package platform

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

type APIError struct {
	Status  int
	Message string
	Field   string
}

func (e *APIError) Error() string { return e.Message }

func BadRequest(field, msg string) *APIError {
	return &APIError{Status: http.StatusBadRequest, Message: msg, Field: field}
}

func Unauthorized(msg string) *APIError {
	return &APIError{Status: http.StatusUnauthorized, Message: msg, Field: "message"}
}

func NotFound(field, msg string) *APIError {
	return &APIError{Status: http.StatusNotFound, Message: msg, Field: field}
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func WriteError(w http.ResponseWriter, err error, defaultField string) {
	var api *APIError
	if errors.As(err, &api) {
		field := api.Field
		if field == "" {
			field = defaultField
		}
		WriteJSON(w, api.Status, map[string]string{field: api.Message})
		return
	}
	WriteJSON(w, http.StatusInternalServerError, map[string]string{defaultField: "Internal server error"})
}

func DecodeJSON(r *http.Request, dest any) error {
	if r.Body == nil {
		return BadRequest("message", "request body is required")
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	// Frontend payloads match the known fields. Unknown fields are rejected so typos fail closed.
	// Re-enable if a client starts sending extras; the React app does not.
	if err := dec.Decode(dest); err != nil {
		if errors.Is(err, io.EOF) {
			return BadRequest("message", "request body is required")
		}
		return BadRequest("message", "invalid JSON body")
	}
	return nil
}

// DecodeJSONLenient ignores unknown fields, matching Jackson.
func DecodeJSONLenient(r *http.Request, dest any) error {
	if r.Body == nil {
		return io.EOF
	}
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dest); err != nil {
		return err
	}
	return nil
}

func CORS(origins []string, next http.Handler) http.Handler {
	allowed := map[string]struct{}{}
	for _, o := range origins {
		allowed[o] = struct{}{}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if _, ok := allowed[origin]; ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "accept, authorization, content-type, x-requested-with")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func BearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) < 7 || !strings.EqualFold(h[:7], "Bearer ") {
		return ""
	}
	return strings.TrimSpace(h[7:])
}

func QueryInt(r *http.Request, name string, def int) int {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return def
	}
	n := 0
	sign := 1
	for i, c := range raw {
		if i == 0 && c == '-' {
			sign = -1
			continue
		}
		if c < '0' || c > '9' {
			return def
		}
		n = n*10 + int(c-'0')
	}
	return n * sign
}
