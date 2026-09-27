// Package configuration loads process settings and the MySQL connection.
package configuration

import (
	"os"
	"strconv"
	"strings"
)

// Config is the process configuration for the content service.
type Config struct {
	Port           string
	DBUser         string
	DBPassword     string
	DBHost         string
	DBPort         string
	DBName         string
	DBSSLMode      string
	JWTSecret      string
	JWTIssuer      string
	JWTLifespan    int64
	JWTKeyID       string
	CORSOrigins    []string
	AuthServiceURL string
}

const (
	defaultJWTLifespan       = "3600"
	defaultCORSOrigins       = "http://localhost:5173,http://localhost:3000"
	defaultDBHost            = "localhost"
	defaultDBPort            = "3306"
	defaultDBName            = "react_cms"
	defaultDBSSLMode         = "PREFERRED"
	defaultJWTSecret         = "change-me"
	defaultJWTIssuer         = "https://react-cms.local/auth"
	defaultAuthServiceURL    = "http://localhost:8081"
	jwtKeyID                 = "react-cms-hs256"
	fallbackJWTLifespanValue = 3600
)

// LoadConfig reads environment variables and falls back to the service port.
func LoadConfig(defaultPort string) Config {
	LoadDotEnv(".env")
	var port string
	port = os.Getenv("PORT")
	if port == "" {
		port = getenv("HTTP_PORT", defaultPort)
	}
	var lifespan int64
	var err error
	lifespan, err = strconv.ParseInt(getenv("JWT_LIFESPAN", defaultJWTLifespan), 10, 64)
	if err != nil || lifespan <= 0 {
		lifespan = fallbackJWTLifespanValue
	}
	var origins []string
	origins = splitCSV(getenv("CORS_ORIGINS", defaultCORSOrigins))
	return Config{
		Port:           port,
		DBUser:         os.Getenv("DB_USER"),
		DBPassword:     os.Getenv("DB_PASSWORD"),
		DBHost:         getenv("DB_HOST", defaultDBHost),
		DBPort:         getenv("DB_PORT", defaultDBPort),
		DBName:         getenv("DB_NAME", defaultDBName),
		DBSSLMode:      getenv("DB_SSL_MODE", defaultDBSSLMode),
		JWTSecret:      getenv("JWT_SECRET", defaultJWTSecret),
		JWTIssuer:      getenv("JWT_ISSUER", defaultJWTIssuer),
		JWTLifespan:    lifespan,
		JWTKeyID:       jwtKeyID,
		CORSOrigins:    origins,
		AuthServiceURL: strings.TrimRight(getenv("AUTH_SERVICE_URL", defaultAuthServiceURL), "/"),
	}
}

func getenv(key, fallback string) string {
	var value string
	value = os.Getenv(key)
	if value != "" {
		return value
	}
	return fallback
}

func splitCSV(raw string) []string {
	var parts []string
	parts = strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
