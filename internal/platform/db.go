package platform

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

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
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=UTC&charset=utf8mb4&collation=utf8mb4_unicode_ci&tls=%s&allowNativePasswords=true",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
		url.QueryEscape(tlsMode),
	)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	return db, nil
}
