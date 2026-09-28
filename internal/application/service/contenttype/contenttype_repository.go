package contenttype

import (
	"context"

	"react-go-cms-content-service/internal/domain/entity"
)

// Repository is the content-type persistence port.
type Repository interface {
	List(ctx context.Context) ([]entity.ContentType, error)
}
