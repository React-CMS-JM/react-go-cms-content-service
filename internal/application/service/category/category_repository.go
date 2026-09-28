package category

import (
	"context"

	"react-go-cms-content-service/internal/domain/entity"
)

// Repository is the category persistence port.
type Repository interface {
	ListPopular(ctx context.Context, lang string, limit int) ([]entity.Taxonomy, error)
	SearchIDs(ctx context.Context, query string, limit int) ([]int, error)
	ByIDs(ctx context.Context, ids []int, lang string) ([]entity.Taxonomy, error)
	Admin(ctx context.Context, lang string, page, size int, query string) (entity.Page[entity.Taxonomy], error)
	Insert(ctx context.Context, description, lang, name, slug string) (int, error)
	Exists(ctx context.Context, id int) (bool, error)
	UpdateDescription(ctx context.Context, id int, description string) error
	FindTranslation(ctx context.Context, id int, lang string) (int, bool, error)
	InsertTranslation(ctx context.Context, id int, lang, name, slug string) error
	UpdateTranslationName(ctx context.Context, translationID int, name string) error
	UpdateTranslationSlug(ctx context.Context, translationID int, slug string) error
	DeleteRow(ctx context.Context, id int) (bool, error)
	DeleteTranslations(ctx context.Context, id int) error
	DeleteLinks(ctx context.Context, id int) error
	ClearPopular()
	RefreshUsageCounts(ctx context.Context, ids []int) error
}
