package tag

import (
	"context"
	"fmt"

	"react-go-cms-content-service/internal/domain/entity"
)

// ListPopular returns the cached or freshly loaded popular tags.
func (s *Service) ListPopular(ctx context.Context, lang string) ([]entity.Taxonomy, error) {
	var rows []entity.Taxonomy
	var err error
	rows, err = s.repository.ListPopular(ctx, normalizeLang(lang), popularLimit)
	if err != nil {
		return nil, fmt.Errorf("list popular tags: %w", err)
	}
	return rows, nil
}

// Search returns tags whose name or slug matches the query.
func (s *Service) Search(ctx context.Context, query, lang string, limit int) ([]entity.Taxonomy, error) {
	normalized := normalizeSearch(query)
	if normalized == "" {
		return []entity.Taxonomy{}, nil
	}
	if limit <= 0 || limit > maxSearchLimit {
		limit = defaultSearchLimit
	}
	language := normalizeLang(lang)
	var ids []int
	var err error
	ids, err = s.repository.SearchIDs(ctx, normalized, limit)
	if err != nil {
		return nil, fmt.Errorf("search tags: %w", err)
	}
	if len(ids) == 0 {
		return []entity.Taxonomy{}, nil
	}
	var items []entity.Taxonomy
	items, err = s.repository.ByIDs(ctx, ids, language)
	if err != nil {
		return nil, fmt.Errorf("search tags: %w", err)
	}
	byID := map[int]entity.Taxonomy{}
	for _, item := range items {
		byID[item.ID] = item
	}
	ordered := make([]entity.Taxonomy, 0, len(ids))
	for _, id := range ids {
		if item, ok := byID[id]; ok {
			ordered = append(ordered, item)
		}
	}
	sortTaxonomy(ordered)
	if len(ordered) > limit {
		ordered = ordered[:limit]
	}
	return ordered, nil
}

// ByIDs returns the tags for the given ids.
func (s *Service) ByIDs(ctx context.Context, ids []int, lang string) ([]entity.Taxonomy, error) {
	var rows []entity.Taxonomy
	var err error
	rows, err = s.repository.ByIDs(ctx, ids, normalizeLang(lang))
	if err != nil {
		return nil, fmt.Errorf("tags by ids: %w", err)
	}
	return rows, nil
}

// Admin returns one admin page of tags.
func (s *Service) Admin(ctx context.Context, lang string, page, size int, query string) (entity.Page[entity.Taxonomy], error) {
	language := normalizeLang(lang)
	if page < minPageIndex {
		page = minPageIndex
	}
	if size <= 0 {
		size = defaultPageSize
	}
	if size > maxPageSize {
		size = maxPageSize
	}
	var result entity.Page[entity.Taxonomy]
	var err error
	result, err = s.repository.Admin(ctx, language, page, size, normalizeSearch(query))
	if err != nil {
		return entity.Page[entity.Taxonomy]{}, fmt.Errorf("admin tags: %w", err)
	}
	return result, nil
}
