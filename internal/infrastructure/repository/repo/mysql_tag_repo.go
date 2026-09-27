package repo

import (
	"context"
	"database/sql"
	"strings"

	"react-go-cms-content-service/internal/application/service/tag"
	"react-go-cms-content-service/internal/domain/entity"
	"react-go-cms-content-service/internal/infrastructure/cache"
	"react-go-cms-content-service/internal/infrastructure/repository/model"
)

const tagCachePrefix = "tag"

// TagRepository persists tags in MySQL.
type TagRepository struct {
	db      *sql.DB
	popular *cache.Popular
}

// NewTagRepository builds a tag repository with the shared popular cache.
func NewTagRepository(db *sql.DB, popular *cache.Popular) *TagRepository {
	return &TagRepository{db: db, popular: popular}
}

var _ tag.Repository = (*TagRepository)(nil)

// ListPopular returns popular tags, using the injected cache when it is fresh.
func (r *TagRepository) ListPopular(ctx context.Context, lang string, limit int) ([]entity.Taxonomy, error) {
	key := tagCachePrefix + ":" + lang
	snap, ok := r.popular.Get(key)
	if ok {
		return snap.Tags, nil
	}
	var rows []entity.Taxonomy
	var err error
	rows, err = r.list(ctx, lang, limit)
	if err != nil {
		return nil, err
	}
	r.popular.Put(key, cache.Snapshot{Tags: rows, Kind: tagCachePrefix})
	return rows, nil
}

// SearchIDs returns tag ids whose translation name or slug matches the query.
func (r *TagRepository) SearchIDs(ctx context.Context, query string, limit int) ([]int, error) {
	pattern := "%" + strings.ToLower(query) + "%"
	var rows *sql.Rows
	var err error
	rows, err = r.db.QueryContext(ctx, `
		SELECT DISTINCT i.tag_id FROM tag_i18n i
		WHERE LOWER(i.name) LIKE ? OR LOWER(i.slug) LIKE ?
		LIMIT ?`, pattern, pattern, limit*4)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int{}
	for rows.Next() {
		var id int
		err = rows.Scan(&id)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ByIDs loads tags for the given ids.
func (r *TagRepository) ByIDs(ctx context.Context, ids []int, lang string) ([]entity.Taxonomy, error) {
	if len(ids) == 0 {
		return []entity.Taxonomy{}, nil
	}
	args := make([]any, 0, len(ids))
	seen := map[int]struct{}{}
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		args = append(args, id)
	}
	if len(args) == 0 {
		return []entity.Taxonomy{}, nil
	}
	query := `SELECT id, db_description, usage_count FROM tags WHERE id IN (` + placeholders(len(args)) + `)`
	var rows *sql.Rows
	var err error
	rows, err = r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	base := []model.TaxonomyRow{}
	for rows.Next() {
		var row model.TaxonomyRow
		err = rows.Scan(&row.ID, &row.DBDescription, &row.UsageCount)
		if err != nil {
			return nil, err
		}
		base = append(base, row)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return r.hydrate(ctx, lang, base)
}

// Admin returns one admin page. An empty query lists every tag.
func (r *TagRepository) Admin(ctx context.Context, lang string, page, size int, query string) (entity.Page[entity.Taxonomy], error) {
	if query == "" {
		var total int64
		var err error
		err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tags`).Scan(&total)
		if err != nil {
			return entity.Page[entity.Taxonomy]{}, err
		}
		var rows *sql.Rows
		rows, err = r.db.QueryContext(ctx, `SELECT id, db_description, usage_count FROM tags ORDER BY usage_count DESC, updated_at DESC, created_at DESC LIMIT ? OFFSET ?`, size, page*size)
		if err != nil {
			return entity.Page[entity.Taxonomy]{}, err
		}
		defer rows.Close()
		var base []model.TaxonomyRow
		for rows.Next() {
			var row model.TaxonomyRow
			err = rows.Scan(&row.ID, &row.DBDescription, &row.UsageCount)
			if err != nil {
				return entity.Page[entity.Taxonomy]{}, err
			}
			base = append(base, row)
		}
		err = rows.Err()
		if err != nil {
			return entity.Page[entity.Taxonomy]{}, err
		}
		var items []entity.Taxonomy
		items, err = r.hydrate(ctx, lang, base)
		if err != nil {
			return entity.Page[entity.Taxonomy]{}, err
		}
		return entity.Page[entity.Taxonomy]{Items: items, Page: page, Size: size, Total: total}, nil
	}
	pattern := "%" + strings.ToLower(query) + "%"
	var total int64
	var err error
	err = r.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT t.id) FROM tags t JOIN tag_i18n i ON i.tag_id = t.id WHERE LOWER(i.name) LIKE ? OR LOWER(i.slug) LIKE ?`, pattern, pattern).Scan(&total)
	if err != nil {
		return entity.Page[entity.Taxonomy]{}, err
	}
	var rows *sql.Rows
	rows, err = r.db.QueryContext(ctx, `SELECT t.id FROM tags t JOIN tag_i18n i ON i.tag_id = t.id
			WHERE LOWER(i.name) LIKE ? OR LOWER(i.slug) LIKE ?
			GROUP BY t.id, t.usage_count, t.updated_at, t.created_at
			ORDER BY t.usage_count DESC, t.updated_at DESC, t.created_at DESC
			LIMIT ? OFFSET ?`, pattern, pattern, size, page*size)
	if err != nil {
		return entity.Page[entity.Taxonomy]{}, err
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		err = rows.Scan(&id)
		if err != nil {
			return entity.Page[entity.Taxonomy]{}, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	if err != nil {
		return entity.Page[entity.Taxonomy]{}, err
	}
	var items []entity.Taxonomy
	items, err = r.ByIDs(ctx, ids, lang)
	if err != nil {
		return entity.Page[entity.Taxonomy]{}, err
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
	if ordered == nil {
		ordered = []entity.Taxonomy{}
	}
	return entity.Page[entity.Taxonomy]{Items: ordered, Page: page, Size: size, Total: total}, nil
}

// Insert stores a tag and its first translation.
func (r *TagRepository) Insert(ctx context.Context, description, lang, name, slug string) (int, error) {
	var result sql.Result
	var err error
	result, err = r.db.ExecContext(ctx, `INSERT INTO tags (db_description, usage_count) VALUES (?, 0)`, description)
	if err != nil {
		return 0, err
	}
	id64, _ := result.LastInsertId()
	id := int(id64)
	_, err = r.db.ExecContext(ctx, `INSERT INTO tag_i18n (tag_id, language_code, name, slug) VALUES (?, ?, ?, ?)`, id, lang, name, slug)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// Exists reports whether the tag row is present.
func (r *TagRepository) Exists(ctx context.Context, id int) (bool, error) {
	var exists int
	var err error
	err = r.db.QueryRowContext(ctx, `SELECT 1 FROM tags WHERE id = ?`, id).Scan(&exists)
	if isNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// UpdateDescription sets db_description.
func (r *TagRepository) UpdateDescription(ctx context.Context, id int, description string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE tags SET db_description = ? WHERE id = ?`, description, id)
	return err
}

// FindTranslation returns the translation row id for a language.
func (r *TagRepository) FindTranslation(ctx context.Context, id int, lang string) (int, bool, error) {
	var translationID int
	var err error
	err = r.db.QueryRowContext(ctx, `SELECT id FROM tag_i18n WHERE tag_id = ? AND language_code = ?`, id, lang).Scan(&translationID)
	if isNotFound(err) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return translationID, true, nil
}

// InsertTranslation adds a tag translation.
func (r *TagRepository) InsertTranslation(ctx context.Context, id int, lang, name, slug string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO tag_i18n (tag_id, language_code, name, slug) VALUES (?, ?, ?, ?)`, id, lang, name, slug)
	return err
}

// UpdateTranslationName sets the translation name.
func (r *TagRepository) UpdateTranslationName(ctx context.Context, translationID int, name string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE tag_i18n SET name = ? WHERE id = ?`, name, translationID)
	return err
}

// UpdateTranslationSlug sets the translation slug.
func (r *TagRepository) UpdateTranslationSlug(ctx context.Context, translationID int, slug string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE tag_i18n SET slug = ? WHERE id = ?`, slug, translationID)
	return err
}

// DeleteRow deletes the tag. False means the row was already absent.
func (r *TagRepository) DeleteRow(ctx context.Context, id int) (bool, error) {
	var result sql.Result
	var err error
	result, err = r.db.ExecContext(ctx, `DELETE FROM tags WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	affected, _ := result.RowsAffected()
	return affected != 0, nil
}

// DeleteTranslations removes tag translations. Callers ignore the error, matching the previous store.
func (r *TagRepository) DeleteTranslations(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM tag_i18n WHERE tag_id = ?`, id)
	return err
}

// DeleteLinks removes post-tag links. Callers ignore the error, matching the previous store.
func (r *TagRepository) DeleteLinks(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM posts_tags WHERE tag_id = ?`, id)
	return err
}

// ClearPopular drops the shared popular category and tag cache.
func (r *TagRepository) ClearPopular() {
	r.popular.Clear()
}

// RefreshUsageCounts recalculates usage_count and clears the popular cache when ids are present.
func (r *TagRepository) RefreshUsageCounts(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	for _, id := range ids {
		_, err := r.db.ExecContext(ctx, `
			UPDATE tags SET usage_count = (
				SELECT COUNT(*) FROM posts_tags WHERE tag_id = ?
			) WHERE id = ?`, id, id)
		if err != nil {
			return err
		}
	}
	r.popular.Clear()
	return nil
}

func (r *TagRepository) list(ctx context.Context, lang string, limit int) ([]entity.Taxonomy, error) {
	var rows *sql.Rows
	var err error
	rows, err = r.db.QueryContext(ctx, `SELECT id, db_description, usage_count FROM tags ORDER BY usage_count DESC, updated_at DESC, created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	base := []model.TaxonomyRow{}
	for rows.Next() {
		var row model.TaxonomyRow
		err = rows.Scan(&row.ID, &row.DBDescription, &row.UsageCount)
		if err != nil {
			return nil, err
		}
		base = append(base, row)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return r.hydrate(ctx, lang, base)
}

func (r *TagRepository) hydrate(ctx context.Context, lang string, base []model.TaxonomyRow) ([]entity.Taxonomy, error) {
	out := make([]entity.Taxonomy, 0, len(base))
	if len(base) == 0 {
		return out, nil
	}
	ids := make([]any, len(base))
	for index, row := range base {
		ids[index] = row.ID
	}
	query := `SELECT tag_id, language_code, name, slug FROM tag_i18n WHERE tag_id IN (` + placeholders(len(ids)) + `) ORDER BY id`
	var rows *sql.Rows
	var err error
	rows, err = r.db.QueryContext(ctx, query, ids...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type translation struct {
		lang, name, slug string
	}
	byID := map[int][]translation{}
	for rows.Next() {
		var row model.TaxonomyI18nRow
		err = rows.Scan(&row.OwnerID, &row.LanguageCode, &row.Name, &row.Slug)
		if err != nil {
			return nil, err
		}
		byID[row.OwnerID] = append(byID[row.OwnerID], translation{lang: row.LanguageCode, name: row.Name, slug: row.Slug})
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	for _, row := range base {
		item := entity.Taxonomy{
			ID:            row.ID,
			DBDescription: StrPtr(row.DBDescription),
			UsageCount:    row.UsageCount,
			LanguageCode:  lang,
			Name:          "",
			Slug:          "",
		}
		translations := byID[row.ID]
		chosen := -1
		for index, translation := range translations {
			if translation.lang == lang {
				chosen = index
				break
			}
		}
		if chosen < 0 && len(translations) > 0 {
			chosen = 0
		}
		if chosen >= 0 {
			item.LanguageCode = translations[chosen].lang
			item.Name = translations[chosen].name
			item.Slug = translations[chosen].slug
		}
		out = append(out, item)
	}
	return out, nil
}
