package repo

import (
	"context"
	"database/sql"

	"react-go-cms-content-service/internal/application/service/dashboard"
	"react-go-cms-content-service/internal/domain/entity"
	"react-go-cms-content-service/internal/infrastructure/repository/model"
)

// DashboardRepository reads dashboard aggregates from MySQL.
type DashboardRepository struct {
	db *sql.DB
}

// NewDashboardRepository builds a dashboard repository.
func NewDashboardRepository(db *sql.DB) *DashboardRepository {
	return &DashboardRepository{db: db}
}

var _ dashboard.Repository = (*DashboardRepository)(nil)

// CountBySlug counts posts of one content type. A query error yields zero.
func (r *DashboardRepository) CountBySlug(ctx context.Context, slug string) int64 {
	var count int64
	var err error
	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM posts p JOIN content_types ct ON ct.id = p.content_type_id WHERE ct.slug = ?`, slug).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

// PendingComments counts comments whose status is pending. A query error yields zero.
func (r *DashboardRepository) PendingComments(ctx context.Context) int64 {
	var count int64
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM comments WHERE status = 'pending'`).Scan(&count)
	return count
}

// Recent returns the most recently updated posts.
func (r *DashboardRepository) Recent(ctx context.Context, limit int) ([]entity.RecentPost, error) {
	var rows *sql.Rows
	var err error
	rows, err = r.db.QueryContext(ctx, `
		SELECT p.id, p.author_id, p.content_type_id, p.status, p.updated_at
		FROM posts p ORDER BY p.updated_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []entity.RecentPost
	for rows.Next() {
		var row model.RecentPostRow
		err = rows.Scan(&row.ID, &row.AuthorID, &row.ContentTypeID, &row.Status, &row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, entity.RecentPost{
			ID:            row.ID,
			AuthorID:      row.AuthorID,
			ContentTypeID: row.ContentTypeID,
			Status:        row.Status,
			UpdatedAt:     LocalFrom(row.UpdatedAt),
		})
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ContentTypeSlug returns a content type slug, or an empty string when the lookup fails.
func (r *DashboardRepository) ContentTypeSlug(ctx context.Context, typeID int) string {
	var slug string
	_ = r.db.QueryRowContext(ctx, `SELECT slug FROM content_types WHERE id = ?`, typeID).Scan(&slug)
	return slug
}

// Title returns the post title in the requested language, then any language.
func (r *DashboardRepository) Title(ctx context.Context, postID, language string) string {
	var title string
	var err error
	err = r.db.QueryRowContext(ctx, `SELECT title FROM post_i18n WHERE post_id = ? AND language_code = ? LIMIT 1`, postID, language).Scan(&title)
	if isNotFound(err) {
		_ = r.db.QueryRowContext(ctx, `SELECT title FROM post_i18n WHERE post_id = ? ORDER BY id LIMIT 1`, postID).Scan(&title)
	}
	return title
}
