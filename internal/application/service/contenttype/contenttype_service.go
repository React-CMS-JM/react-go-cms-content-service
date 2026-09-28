// Package contenttype applies content-type queries.
package contenttype

// Service coordinates content-type queries.
type Service struct {
	repository Repository
}

// New builds a content-type service.
func New(repository Repository) *Service {
	return &Service{repository: repository}
}
