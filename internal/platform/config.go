package platform

import (
	"os"
	"strconv"
	"strings"
)

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

func LoadConfig(defaultPort string) Config {
	LoadDotEnv(".env")
	port := os.Getenv("PORT")
	if port == "" {
		port = getenv("HTTP_PORT", defaultPort)
	}
	lifespan, err := strconv.ParseInt(getenv("JWT_LIFESPAN", "3600"), 10, 64)
	if err != nil || lifespan <= 0 {
		lifespan = 3600
	}
	origins := splitCSV(getenv("CORS_ORIGINS", "http://localhost:5173,http://localhost:3000"))
	return Config{
		Port:           port,
		DBUser:         os.Getenv("DB_USER"),
		DBPassword:     os.Getenv("DB_PASSWORD"),
		DBHost:         getenv("DB_HOST", "localhost"),
		DBPort:         getenv("DB_PORT", "3306"),
		DBName:         getenv("DB_NAME", "react_cms"),
		DBSSLMode:      getenv("DB_SSL_MODE", "PREFERRED"),
		JWTSecret:      getenv("JWT_SECRET", "change-me"),
		JWTIssuer:      getenv("JWT_ISSUER", "https://react-cms.local/auth"),
		JWTLifespan:    lifespan,
		JWTKeyID:       "react-cms-hs256",
		CORSOrigins:    origins,
		AuthServiceURL: strings.TrimRight(getenv("AUTH_SERVICE_URL", "http://localhost:8081"), "/"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
