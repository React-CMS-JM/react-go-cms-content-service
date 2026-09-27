package uistring

import (
	"context"
	"fmt"

	"react-go-cms-content-service/internal/domain/entity"
)

// List returns UI strings for one language and optional component.
func (s *Service) List(ctx context.Context, lang, component string) ([]entity.UIString, error) {
	var rows []entity.UIString
	var err error
	rows, err = s.repository.List(ctx, normalizeLang(lang), component)
	if err != nil {
		return nil, fmt.Errorf("list ui strings: %w", err)
	}
	if rows == nil {
		return []entity.UIString{}, nil
	}
	return rows, nil
}
