// Package dashboard applies the admin dashboard query.
package dashboard

import (
	"context"
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

// Service coordinates the dashboard query.
type Service struct {
	repository Repository
	users      UserClient
}

// New builds a dashboard service.
func New(repository Repository, users UserClient) *Service {
	return &Service{repository: repository, users: users}
}
