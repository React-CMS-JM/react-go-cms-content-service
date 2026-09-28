package dashboard

import (
	"context"

	"react-go-cms-content-service/internal/domain/entity"
)

// Repository is the dashboard persistence port.
type Repository interface {
	CountBySlug(ctx context.Context, slug string) int64
	PendingComments(ctx context.Context) int64
	Recent(ctx context.Context, limit int) ([]entity.RecentPost, error)
	ContentTypeSlug(ctx context.Context, typeID int) string
	Title(ctx context.Context, postID, language string) string
}
