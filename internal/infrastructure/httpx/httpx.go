// Package httpx writes JSON responses and applies CORS for the content service.
package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

// APIError is an HTTP failure with a response field name.
type APIError struct {
	Status  int
	Message string
	Field   string
}

// Error returns the response message.
func (e *APIError) Error() string { return e.Message }

// BadRequest builds a 400 API error.
func BadRequest(field, msg string) *APIError {
	return &APIError{Status: http.StatusBadRequest, Message: msg, Field: field}
}

// Unauthorized builds a 401 API error. The JSON field stays "message".
func Unauthorized(msg string) *APIError {
	return &APIError{Status: http.StatusUnauthorized, Message: msg, Field: "message"}
}

// NotFound builds a 404 API error.
func NotFound(field, msg string) *APIError {
	return &APIError{Status: http.StatusNotFound, Message: msg, Field: field}
}

// WriteJSON writes a JSON body with HTML escaping disabled.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

// WriteNoContent writes HTTP 204.
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// WriteError writes an APIError body, or a generic 500.
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

// DecodeJSON decodes a body and rejects unknown fields.
func DecodeJSON(r *http.Request, dest any) error {
	if r.Body == nil {
		return BadRequest("message", "request body is required")
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var err error
	err = dec.Decode(dest)
	if err != nil {
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
	var err error
	err = dec.Decode(dest)
	if err != nil {
		return err
	}
	return nil
}

// CORS allows the configured browser origins.
func CORS(origins []string, next http.Handler) http.Handler {
	allowed := map[string]struct{}{}
	for _, origin := range origins {
		allowed[origin] = struct{}{}
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

// BearerToken returns the bearer token, or an empty string when the header is absent.
func BearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if len(header) < 7 || !strings.EqualFold(header[:7], "Bearer ") {
		return ""
	}
	return strings.TrimSpace(header[7:])
}

// QueryInt reads an integer query parameter, or def when it is missing or invalid.
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
