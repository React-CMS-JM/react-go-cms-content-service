// Package tag applies tag use cases.
package tag

import (
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
	notFoundLabel       = "Tag"
)

var nonWord = regexp.MustCompile(`[^\w\s-]`)
var spaces = regexp.MustCompile(`[\s_-]+`)
var edgeDash = regexp.MustCompile(`^-+|-+$`)

// Service coordinates tag commands and queries.
type Service struct {
	repository Repository
}

// New builds a tag service.
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
