// Package settings applies site settings, home hero, home sections, and main menu use cases.
package settings

import (
	"strings"
)

const (
	defaultLanguage     = "en"
	defaultPostsPerPage = 9
	defaultSiteName     = "React CMS"
	heroHref            = "/"
	heroTextColor       = "#ffffff"
	heroBackground      = "#6366f1"
	minItemLimit        = 1
	maxItemLimit        = 50
	servicesLimit       = 5
	productsLimit       = 6
	blogLimit           = 9
	coursesLimit        = 3
	fallbackLimit       = 6
)

// Service coordinates settings commands and queries.
type Service struct {
	repository Repository
}

// New builds a settings service.
func New(repository Repository) *Service {
	return &Service{repository: repository}
}

func normalizeLang(lang string) string {
	lang = strings.TrimSpace(strings.ToLower(lang))
	if lang == "" {
		return defaultLanguage
	}
	return lang
}

func defaultSectionLimit(section string) int {
	switch strings.ToLower(strings.TrimSpace(section)) {
	case "services":
		return servicesLimit
	case "products":
		return productsLimit
	case "blog":
		return blogLimit
	case "courses":
		return coursesLimit
	default:
		return fallbackLimit
	}
}

func normalizeLimit(value int) int {
	if value < minItemLimit {
		return minItemLimit
	}
	if value > maxItemLimit {
		return maxItemLimit
	}
	return value
}

func normalizeLimitPtr(value *int, fallback int) int {
	if value == nil {
		return normalizeLimit(fallback)
	}
	return normalizeLimit(*value)
}
