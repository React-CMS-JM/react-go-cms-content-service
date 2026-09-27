// Package identity generates content-service identifiers.
package identity

import "crypto/rand"

func randRead(b []byte) (int, error) {
	return rand.Read(b)
}

// NewUUID returns a random UUID string.
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
