// Package dashboard applies the admin dashboard query.
package dashboard

import (
	"context"

	"react-go-cms-content-service/internal/domain/entity"
)

const (
	defaultRecentLimit = 6
	maxRecentLimit     = 20
	defaultLanguage    = "en"
	fallbackTypeSlug   = "post"
)

// UserClient loads display names for author ids from the auth service.
type UserClient interface {
	AuthorNames(ctx context.Context, ids []string, authorization string) map[string]string
}

// Repository is the dashboard persistence port.
type Repository interface {
	CountBySlug(ctx context.Context, slug string) int64
	PendingComments(ctx context.Context) int64
	Recent(ctx context.Context, limit int) ([]entity.RecentPost, error)
	ContentTypeSlug(ctx context.Context, typeID int) string
	Title(ctx context.Context, postID, language string) string
}

// Service coordinates the dashboard query.
type Service struct {
	repository Repository
	users      UserClient
}

// New builds a dashboard service.
func New(repository Repository, users UserClient) *Service {
	return &Service{repository: repository, users: users}
}
