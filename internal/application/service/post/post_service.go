// Package post applies post use cases.
package post

import (
	"context"
	"regexp"
	"strings"
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
