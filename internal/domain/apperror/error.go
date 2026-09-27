// Package apperror holds sentinel domain errors for the content service.
package apperror

// Kind classifies a domain error for HTTP mapping.
type Kind int

const (
	// KindInvalid means the caller sent a rule-breaking request.
	KindInvalid Kind = iota + 1
	// KindNotFound means the requested resource does not exist for this caller.
	KindNotFound
)

// Error is a domain failure that handlers map to an HTTP status.
type Error struct {
	Kind    Kind
	Message string
}

// Error returns the client-facing message.
func (e *Error) Error() string {
	return e.Message
}

// Is matches sentinels by kind so errors.Is works after %w wrapping.
func (e *Error) Is(target error) bool {
	other, ok := target.(*Error)
	if !ok {
		return false
	}
	if other.Message != "" && other.Message != e.Message {
		return false
	}
	return e.Kind == other.Kind
}

// ErrInvalid matches any invalid-input domain error.
var ErrInvalid = &Error{Kind: KindInvalid}

// ErrNotFound matches any missing-resource domain error.
var ErrNotFound = &Error{Kind: KindNotFound}

// Invalid builds a bad-input error with a client-facing message.
func Invalid(message string) *Error {
	return &Error{Kind: KindInvalid, Message: message}
}

// NotFound builds a missing-resource error with a client-facing message.
func NotFound(message string) *Error {
	return &Error{Kind: KindNotFound, Message: message}
}
