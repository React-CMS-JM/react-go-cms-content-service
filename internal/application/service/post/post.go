// Package post applies post use cases.
package post

import (
	"context"
	"regexp"
	"strings"

	"react-go-cms-content-service/internal/domain/entity"
)

const (
	statusPublished = "published"
	statusDraft     = "draft"
	accessPublic    = "public"
	defaultLanguage = "en"
	minPageIndex    = 0
	minPageSize     = 1
	maxPageSize     = 100
	courseSlug      = "course"
)

var nonWord = regexp.MustCompile(`[^\w\s-]`)
var spaces = regexp.MustCompile(`[\s_-]+`)
var edgeDash = regexp.MustCompile(`^-+|-+$`)

// ListFilter selects one page of posts.
type ListFilter struct {
	TypeSlug string
	Status   string
	Page     int
	Size     int
}

// InsertInput is a post ready to insert.
type InsertInput struct {
	AuthorID         string
	ContentTypeID    int
	FeaturedImageURL *string
	AccessLevel      string
	Status           string
	PublishedAt      *string
	LanguageCode     string
	Title            string
	Slug             string
	Content          string
	Excerpt          *string
	MetaTitle        *string
	MetaDescription  *string
	CategoryIDs      []int
	TagIDs           []int
}

// UpdateInput is a partial post change. LanguageCode is already normalized.
type UpdateInput struct {
	ContentTypeID    *int
	FeaturedImageURL *string
	AccessLevel      *string
	Status           *string
	CategoryIDs      *[]int
	TagIDs           *[]int
	LanguageCode     string
	Title            *string
	Slug             *string
	Content          *string
	Excerpt          *string
	MetaTitle        *string
	MetaDescription  *string
}

// UsageRefresher recalculates taxonomy usage after post link changes.
type UsageRefresher interface {
	RefreshUsageCounts(ctx context.Context, ids []int) error
}

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

// Service coordinates post commands and queries.
type Service struct {
	repository Repository
	categories UsageRefresher
	tags       UsageRefresher
}

// New builds a post service.
func New(repository Repository, categories, tags UsageRefresher) *Service {
	return &Service{repository: repository, categories: categories, tags: tags}
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

func ownedContentType(slug string) bool {
	switch slug {
	case "post", "page", "service", "product":
		return true
	default:
		return false
	}
}

func emptyString() *string {
	value := ""
	return &value
}
