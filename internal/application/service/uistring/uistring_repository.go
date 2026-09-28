package uistring

import (
	"context"

	"react-go-cms-content-service/internal/domain/entity"
)

// Repository is the UI-string persistence port.
type Repository interface {
	List(ctx context.Context, language, component string) ([]entity.UIString, error)
	Upsert(ctx context.Context, stringKey, language, value string) (entity.UIString, error)
}
