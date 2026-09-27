// Package contenttype applies content-type queries.
package contenttype

import (
	"context"

	"react-go-cms-content-service/internal/domain/entity"
)

// Repository is the content-type persistence port.
type Repository interface {
	List(ctx context.Context) ([]entity.ContentType, error)
}

// Service coordinates content-type queries.
type Service struct {
	repository Repository
}

// New builds a content-type service.
func New(repository Repository) *Service {
	return &Service{repository: repository}
}
