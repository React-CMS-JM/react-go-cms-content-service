package repo

import (
	"database/sql"
	"errors"
	"strings"
)

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	repeated := strings.Repeat("?,", n)
	return repeated[:len(repeated)-1]
}

func isNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
