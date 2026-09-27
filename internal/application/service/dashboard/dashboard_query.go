package dashboard

import (
	"context"
	"fmt"
	"strings"

	"react-go-cms-content-service/internal/domain/entity"
)

// Load builds the admin dashboard for one language.
func (s *Service) Load(ctx context.Context, lang string, recentLimit int, authorization string) (entity.Dashboard, error) {
	language := normalizeLang(lang)
	if recentLimit <= 0 {
		recentLimit = defaultRecentLimit
	}
	if recentLimit > maxRecentLimit {
		recentLimit = maxRecentLimit
	}
	dashboard := entity.Dashboard{Recent: []entity.Recent{}}
	dashboard.Counts.Post = s.repository.CountBySlug(ctx, "post")
	dashboard.Counts.Page = s.repository.CountBySlug(ctx, "page")
	dashboard.Counts.Service = s.repository.CountBySlug(ctx, "service")
	dashboard.Counts.Product = s.repository.CountBySlug(ctx, "product")
	dashboard.PendingComments = s.repository.PendingComments(ctx)
	var rows []entity.RecentPost
	var err error
	rows, err = s.repository.Recent(ctx, recentLimit)
	if err != nil {
		return dashboard, fmt.Errorf("load dashboard: %w", err)
	}
	authors := []string{}
	for _, row := range rows {
		if strings.TrimSpace(row.AuthorID) != "" {
			authors = append(authors, row.AuthorID)
		}
	}
	names := s.users.AuthorNames(ctx, authors, authorization)
	for _, row := range rows {
		slug := s.repository.ContentTypeSlug(ctx, row.ContentTypeID)
		if slug == "" {
			slug = fallbackTypeSlug
		}
		dashboard.Recent = append(dashboard.Recent, entity.Recent{
			ID:              row.ID,
			Title:           s.repository.Title(ctx, row.ID, language),
			AuthorID:        row.AuthorID,
			AuthorName:      names[row.AuthorID],
			Status:          row.Status,
			ContentTypeSlug: slug,
			UpdatedAt:       row.UpdatedAt,
		})
	}
	return dashboard, nil
}

func normalizeLang(lang string) string {
	lang = strings.TrimSpace(strings.ToLower(lang))
	if lang == "" {
		return defaultLanguage
	}
	return lang
}
