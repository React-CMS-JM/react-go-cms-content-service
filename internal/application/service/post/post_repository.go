package post

import (
	"context"

	"react-go-cms-content-service/internal/domain/entity"
)

// Repository is the post persistence port.
type Repository interface {
	List(ctx context.Context, filter ListFilter) ([]entity.PostRecord, int64, error)
	FindByID(ctx context.Context, id string) (entity.PostRecord, error)
	FindIDBySlug(ctx context.Context, slug, language string) (string, error)
	ContentTypeSlug(ctx context.Context, typeID int) (string, error)
	FindContentType(ctx context.Context, id *int, slug *string) (int, string, error)
	LoadDetails(ctx context.Context, records []entity.PostRecord, withMetadata bool) ([]entity.PostRecord, error)
	Insert(ctx context.Context, input InsertInput) (string, []int, []int, error)
	Update(ctx context.Context, id string, input UpdateInput) ([]int, []int, error)
	ApplyStatus(ctx context.Context, id, status string) error
	IncrementView(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) ([]int, []int, error)
	ListMetadata(ctx context.Context, id string) ([]entity.Metadata, error)
	ReplaceMetadata(ctx context.Context, id string, entries []entity.MetadataInput) error
}
