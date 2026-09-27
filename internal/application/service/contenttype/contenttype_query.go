package contenttype

import (
	"context"
	"fmt"

	"react-go-cms-content-service/internal/domain/entity"
)

// List returns every content type ordered by id.
func (s *Service) List(ctx context.Context) ([]entity.ContentType, error) {
	var rows []entity.ContentType
	var err error
	rows, err = s.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list content types: %w", err)
	}
	if rows == nil {
		return []entity.ContentType{}, nil
	}
	return rows, nil
}
