// Package uistring applies UI-string use cases.
package uistring

import (
	"strings"
)

const defaultLanguage = "en"

// Service coordinates UI-string commands and queries.
type Service struct {
	repository Repository
}

// New builds a UI-string service.
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
