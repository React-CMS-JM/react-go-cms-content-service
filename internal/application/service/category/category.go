// Package category applies category use cases.
package category

import (
	"context"
	"regexp"
	"strings"

	"react-go-cms-content-service/internal/domain/entity"
)

const (
	defaultLanguage     = "en"
	minPageIndex        = 0
	defaultPageSize     = 10
	maxPageSize         = 50
	defaultSearchLimit  = 20
	maxSearchLimit      = 20
	minSearchCompactLen = 2
	popularLimit        = 100
	notFoundLabel       = "Category"
)

var nonWord = regexp.MustCompile(`[^\w\s-]`)
var spaces = regexp.MustCompile(`[\s_-]+`)
var edgeDash = regexp.MustCompile(`^-+|-+$`)

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

// Service coordinates category commands and queries.
type Service struct {
	repository Repository
}

// New builds a category service.
func New(repository Repository) *Service {
	return &Service{repository: repository}
}

func normalizeLang(lang string) string {
	lang = strings.TrimSpace(strings.ToLower(lang))
	if lang == "" {
		return defaultLanguage
	}
	return lang
}

func slugify(text string) string {
	slug := strings.ToLower(strings.TrimSpace(text))
	slug = nonWord.ReplaceAllString(slug, "")
	slug = spaces.ReplaceAllString(slug, "-")
	return edgeDash.ReplaceAllString(slug, "")
}

func normalizeSearch(query string) string {
	query = strings.TrimSpace(query)
	compact := strings.ReplaceAll(query, " ", "")
	if len(compact) < minSearchCompactLen {
		return ""
	}
	return query
}

func sortTaxonomy(items []entity.Taxonomy) {
	for index := 1; index < len(items); index++ {
		cursor := index
		for cursor > 0 && taxLess(items[cursor], items[cursor-1]) {
			items[cursor], items[cursor-1] = items[cursor-1], items[cursor]
			cursor--
		}
	}
}

func taxLess(left, right entity.Taxonomy) bool {
	if left.UsageCount != right.UsageCount {
		return left.UsageCount > right.UsageCount
	}
	return left.ID > right.ID
}
