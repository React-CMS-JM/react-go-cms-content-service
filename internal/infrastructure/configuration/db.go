package configuration

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

const (
	maxOpenConns = 20
	maxIdleConns = 5
)

// OpenDB opens the MySQL pool using the content service DSN.
func OpenDB(cfg Config) (*sql.DB, error) {
	tlsMode := "preferred"
	switch strings.ToUpper(cfg.DBSSLMode) {
	case "VERIFY_IDENTITY", "REQUIRED", "TRUE":
		tlsMode = "true"
	case "DISABLED", "FALSE":
		tlsMode = "false"
	case "SKIP_VERIFY":
		tlsMode = "skip-verify"
	case "PREFERRED", "":
		tlsMode = "preferred"
	default:
		tlsMode = strings.ToLower(cfg.DBSSLMode)
	}
	var dsn string
	dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=UTC&charset=utf8mb4&collation=utf8mb4_unicode_ci&tls=%s&allowNativePasswords=true",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
		url.QueryEscape(tlsMode),
	)
	var db *sql.DB
	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	return db, nil
}
