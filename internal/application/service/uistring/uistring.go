// Package uistring applies UI-string use cases.
package uistring

import (
	"context"
	"strings"

	"react-go-cms-content-service/internal/domain/entity"
)

const defaultLanguage = "en"

// Repository is the UI-string persistence port.
type Repository interface {
	List(ctx context.Context, language, component string) ([]entity.UIString, error)
	Upsert(ctx context.Context, stringKey, language, value string) (entity.UIString, error)
}

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
