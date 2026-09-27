package handler

import (
	"strconv"
	"strings"
	"time"
)

// InstantJSON matches Jackson's ISO-8601 Instant (trailing Z).
type InstantJSON struct {
	Time  time.Time
	Valid bool
}

// MarshalJSON emits null when the instant is missing.
func (t InstantJSON) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return []byte("null"), nil
	}
	return []byte(strconv.Quote(formatInstant(t.Time))), nil
}

// LocalJSON matches Jackson LocalDateTime: no timezone suffix.
type LocalJSON struct {
	Time  time.Time
	Valid bool
}

// MarshalJSON emits null when the local datetime is missing.
func (t LocalJSON) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return []byte("null"), nil
	}
	return []byte(strconv.Quote(formatLocal(t.Time))), nil
}

func formatInstant(value time.Time) string {
	value = value.UTC()
	if value.Nanosecond() == 0 {
		return value.Format("2006-01-02T15:04:05Z")
	}
	formatted := value.Format("2006-01-02T15:04:05.000000000Z")
	head, _, _ := strings.Cut(formatted, "Z")
	head = strings.TrimRight(head, "0")
	return head + "Z"
}

func formatLocal(value time.Time) string {
	value = value.In(time.UTC)
	if value.Nanosecond() == 0 {
		return value.Format("2006-01-02T15:04:05")
	}
	formatted := value.Format("2006-01-02T15:04:05.000000000")
	return strings.TrimRight(formatted, "0")
}

func toLocal(value *time.Time) LocalJSON {
	if value == nil {
		return LocalJSON{}
	}
	return LocalJSON{Time: *value, Valid: true}
}
