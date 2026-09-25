package platform

import (
	"database/sql"
	"strconv"
	"strings"
	"time"
)

// InstantJSON matches Jackson's ISO-8601 Instant (trailing Z).
type InstantJSON struct {
	Time  time.Time
	Valid bool
}

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

func (t LocalJSON) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return []byte("null"), nil
	}
	return []byte(strconv.Quote(formatLocal(t.Time))), nil
}

func formatInstant(t time.Time) string {
	t = t.UTC()
	if t.Nanosecond() == 0 {
		return t.Format("2006-01-02T15:04:05Z")
	}
	s := t.Format("2006-01-02T15:04:05.000000000Z")
	head, _, _ := strings.Cut(s, "Z")
	head = strings.TrimRight(head, "0")
	return head + "Z"
}

func formatLocal(t time.Time) string {
	t = t.In(time.UTC)
	if t.Nanosecond() == 0 {
		return t.Format("2006-01-02T15:04:05")
	}
	s := t.Format("2006-01-02T15:04:05.000000000")
	return strings.TrimRight(s, "0")
}

func InstantFrom(nt sql.NullTime) InstantJSON {
	if !nt.Valid {
		return InstantJSON{}
	}
	return InstantJSON{Time: nt.Time, Valid: true}
}

func LocalFrom(nt sql.NullTime) LocalJSON {
	if !nt.Valid {
		return LocalJSON{}
	}
	return LocalJSON{Time: nt.Time, Valid: true}
}

func StrPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	s := ns.String
	return &s
}

func NewUUID() string {
	var b [16]byte
	_, _ = randRead(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	const hex = "0123456789abcdef"
	out := make([]byte, 36)
	pos := 0
	for i, c := range b {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			out[pos] = '-'
			pos++
		}
		out[pos] = hex[c>>4]
		out[pos+1] = hex[c&0x0f]
		pos += 2
	}
	return string(out)
}
