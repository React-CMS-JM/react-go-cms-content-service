package uistring

import (
	"context"
	"fmt"
	"strings"

	"react-go-cms-content-service/internal/domain/apperror"
	"react-go-cms-content-service/internal/domain/entity"
)

// Upsert writes one UI string. A nil value is rejected before the language check.
func (s *Service) Upsert(ctx context.Context, stringKey, languageCode string, value *string) (entity.UIString, error) {
	if value == nil {
		return entity.UIString{}, fmt.Errorf("upsert ui string: %w", apperror.Invalid("stringValue is required"))
	}
	if strings.TrimSpace(languageCode) == "" {
		return entity.UIString{}, fmt.Errorf("upsert ui string: %w", apperror.Invalid("languageCode is required"))
	}
	var item entity.UIString
	var err error
	item, err = s.repository.Upsert(ctx, stringKey, normalizeLang(languageCode), *value)
	if err != nil {
		return entity.UIString{}, fmt.Errorf("upsert ui string: %w", err)
	}
	return item, nil
}

// Patch upserts each item. A nil slice is rejected.
func (s *Service) Patch(ctx context.Context, items []entity.UIStringWrite) ([]entity.UIString, error) {
	if items == nil {
		return nil, fmt.Errorf("patch ui strings: %w", apperror.Invalid("items are required"))
	}
	out := []entity.UIString{}
	for _, item := range items {
		value := item.StringValue
		var row entity.UIString
		var err error
		row, err = s.Upsert(ctx, item.StringKey, item.LanguageCode, &value)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, nil
}
