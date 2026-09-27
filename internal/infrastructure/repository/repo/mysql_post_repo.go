package repo

import (
	"context"
	"database/sql"
	"strings"

	"react-go-cms-content-service/internal/application/service/post"
	"react-go-cms-content-service/internal/domain/apperror"
	"react-go-cms-content-service/internal/domain/entity"
	"react-go-cms-content-service/internal/infrastructure/identity"
	"react-go-cms-content-service/internal/infrastructure/repository/model"
)

const postCols = `p.id, p.author_id, p.content_type_id, p.featured_image_url, p.access_level, p.status, COALESCE(p.view_count,0), p.published_at, p.created_at, p.updated_at`

// PostRepository persists posts in MySQL.
type PostRepository struct {
	db *sql.DB
}

// NewPostRepository builds a post repository.
func NewPostRepository(db *sql.DB) *PostRepository {
	return &PostRepository{db: db}
}

var _ post.Repository = (*PostRepository)(nil)

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// List returns one page of owned posts and the matching total.
func (r *PostRepository) List(ctx context.Context, filter post.ListFilter) ([]entity.PostRecord, int64, error) {
	where := ` FROM posts p JOIN content_types ct ON p.content_type_id = ct.id WHERE ct.slug <> 'course'`
	args := []any{}
	if filter.TypeSlug != "" {
		where += ` AND ct.slug = ?`
		args = append(args, filter.TypeSlug)
	} else {
		where += ` AND ct.slug IN ('post','page','service','product')`
	}
	if filter.Status != "" {
		where += ` AND p.status = ?`
		args = append(args, filter.Status)
	}
	var total int64
	var err error
	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*)`+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	query := `SELECT ` + postCols + where + ` ORDER BY p.created_at DESC LIMIT ? OFFSET ?`
	args = append(args, filter.Size, filter.Page*filter.Size)
	var rows *sql.Rows
	rows, err = r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	posts := []entity.PostRecord{}
	for rows.Next() {
		var record entity.PostRecord
		record, err = scanPost(rows)
		if err != nil {
			return nil, 0, err
		}
		posts = append(posts, record)
	}
	err = rows.Err()
	if err != nil {
		return nil, 0, err
	}
	return posts, total, nil
}

// FindByID loads one post row.
func (r *PostRepository) FindByID(ctx context.Context, id string) (entity.PostRecord, error) {
	var record entity.PostRecord
	var err error
	record, err = scanPost(r.db.QueryRowContext(ctx, `SELECT `+postCols+` FROM posts p WHERE p.id = ?`, id))
	if isNotFound(err) {
		return entity.PostRecord{}, apperror.NotFound("Post not found: " + id)
	}
	return record, err
}

// FindIDBySlug resolves a post id from a slug, preferring the requested language.
func (r *PostRepository) FindIDBySlug(ctx context.Context, slug, language string) (string, error) {
	var postID string
	var err error
	err = r.db.QueryRowContext(ctx, `SELECT post_id FROM post_i18n WHERE slug = ? AND language_code = ? ORDER BY id LIMIT 1`, slug, language).Scan(&postID)
	if isNotFound(err) {
		err = r.db.QueryRowContext(ctx, `SELECT post_id FROM post_i18n WHERE slug = ? ORDER BY id LIMIT 1`, slug).Scan(&postID)
	}
	if isNotFound(err) {
		return "", apperror.NotFound("Post not found for slug: " + slug)
	}
	return postID, err
}

// ContentTypeSlug returns the slug for a content type id.
func (r *PostRepository) ContentTypeSlug(ctx context.Context, typeID int) (string, error) {
	var slug string
	var err error
	err = r.db.QueryRowContext(ctx, `SELECT slug FROM content_types WHERE id = ?`, typeID).Scan(&slug)
	return slug, err
}

// FindContentType loads a content type by id, or by slug when id is absent.
func (r *PostRepository) FindContentType(ctx context.Context, id *int, slug *string) (int, string, error) {
	var typeID int
	var typeSlug string
	var err error
	if id != nil {
		err = r.db.QueryRowContext(ctx, `SELECT id, slug FROM content_types WHERE id = ?`, *id).Scan(&typeID, &typeSlug)
	} else if slug != nil && strings.TrimSpace(*slug) != "" {
		err = r.db.QueryRowContext(ctx, `SELECT id, slug FROM content_types WHERE slug = ?`, strings.ToLower(strings.TrimSpace(*slug))).Scan(&typeID, &typeSlug)
	} else {
		return 0, "", apperror.Invalid("contentTypeId or contentTypeSlug is required")
	}
	if isNotFound(err) {
		return 0, "", apperror.Invalid("contentTypeId or contentTypeSlug is required")
	}
	return typeID, typeSlug, err
}

// LoadDetails attaches translations, taxonomy links, and optional metadata.
func (r *PostRepository) LoadDetails(ctx context.Context, records []entity.PostRecord, withMetadata bool) ([]entity.PostRecord, error) {
	if len(records) == 0 {
		return records, nil
	}
	ids := make([]any, len(records))
	for index, record := range records {
		ids[index] = record.ID
	}
	var translations map[string][]entity.PostTranslation
	var err error
	translations, err = r.loadI18n(ctx, ids)
	if err != nil {
		return nil, err
	}
	var categories map[string][]int
	var tags map[string][]int
	categories, tags, err = r.loadTaxonomyLinks(ctx, ids)
	if err != nil {
		return nil, err
	}
	var metadata map[string][]entity.Metadata
	if withMetadata {
		metadata, err = r.loadMetadata(ctx, ids)
		if err != nil {
			return nil, err
		}
	}
	for index := range records {
		records[index].Translations = translations[records[index].ID]
		records[index].CategoryIDs = categories[records[index].ID]
		records[index].TagIDs = tags[records[index].ID]
		if withMetadata {
			records[index].Metadata = metadata[records[index].ID]
		}
	}
	return records, nil
}

// Insert stores a post, its first translation, and taxonomy links.
func (r *PostRepository) Insert(ctx context.Context, input post.InsertInput) (string, []int, []int, error) {
	var id string
	id = identity.NewUUID()
	var tx *sql.Tx
	var err error
	tx, err = r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", nil, nil, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO posts (id, author_id, content_type_id, featured_image_url, access_level, status, view_count, published_at)
		VALUES (?, ?, ?, ?, ?, ?, 0, ?)`,
		id, input.AuthorID, input.ContentTypeID, input.FeaturedImageURL, input.AccessLevel, input.Status, input.PublishedAt)
	if err != nil {
		return "", nil, nil, err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO post_i18n (post_id, language_code, title, slug, content, excerpt, meta_title, meta_description)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, input.LanguageCode, input.Title, input.Slug, input.Content, input.Excerpt, input.MetaTitle, input.MetaDescription)
	if err != nil {
		return "", nil, nil, err
	}
	var affectedCategories []int
	var affectedTags []int
	affectedCategories, affectedTags, err = replaceLinks(ctx, tx, id, input.CategoryIDs, input.TagIDs, true, true)
	if err != nil {
		return "", nil, nil, err
	}
	err = tx.Commit()
	if err != nil {
		return "", nil, nil, err
	}
	return id, affectedCategories, affectedTags, nil
}

// Update applies a partial post change inside one transaction.
func (r *PostRepository) Update(ctx context.Context, id string, input post.UpdateInput) ([]int, []int, error) {
	var tx *sql.Tx
	var err error
	tx, err = r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()
	if input.ContentTypeID != nil {
		_, err = tx.ExecContext(ctx, `UPDATE posts SET content_type_id = ? WHERE id = ?`, *input.ContentTypeID, id)
		if err != nil {
			return nil, nil, err
		}
	}
	if input.FeaturedImageURL != nil {
		_, err = tx.ExecContext(ctx, `UPDATE posts SET featured_image_url = ? WHERE id = ?`, *input.FeaturedImageURL, id)
		if err != nil {
			return nil, nil, err
		}
	}
	if input.AccessLevel != nil {
		_, err = tx.ExecContext(ctx, `UPDATE posts SET access_level = ? WHERE id = ?`, *input.AccessLevel, id)
		if err != nil {
			return nil, nil, err
		}
	}
	if input.Status != nil {
		err = applyStatus(ctx, tx, id, *input.Status)
		if err != nil {
			return nil, nil, err
		}
	}
	var affectedCategories []int
	var affectedTags []int
	if input.CategoryIDs != nil || input.TagIDs != nil {
		var categories []int
		var tags []int
		replaceCategories, replaceTags := false, false
		if input.CategoryIDs != nil {
			categories = *input.CategoryIDs
			replaceCategories = true
		}
		if input.TagIDs != nil {
			tags = *input.TagIDs
			replaceTags = true
		}
		affectedCategories, affectedTags, err = replaceLinks(ctx, tx, id, categories, tags, replaceCategories, replaceTags)
		if err != nil {
			return nil, nil, err
		}
	}
	hasTranslation := input.Title != nil || input.Slug != nil || input.Content != nil || input.Excerpt != nil || input.MetaTitle != nil || input.MetaDescription != nil
	if hasTranslation {
		var existing int
		err = tx.QueryRowContext(ctx, `SELECT id FROM post_i18n WHERE post_id = ? AND language_code = ?`, id, input.LanguageCode).Scan(&existing)
		if isNotFound(err) {
			title := ""
			if input.Title != nil {
				title = *input.Title
			}
			slug := slugify(title)
			if input.Slug != nil {
				slug = *input.Slug
			}
			content := ""
			if input.Content != nil {
				content = *input.Content
			}
			_, err = tx.ExecContext(ctx, `
				INSERT INTO post_i18n (post_id, language_code, title, slug, content, excerpt, meta_title, meta_description)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				id, input.LanguageCode, title, slug, content, input.Excerpt, input.MetaTitle, input.MetaDescription)
			if err != nil {
				return nil, nil, err
			}
		} else if err != nil {
			return nil, nil, err
		} else {
			err = updatePostTranslation(ctx, tx, existing, input)
			if err != nil {
				return nil, nil, err
			}
		}
	}
	err = tx.Commit()
	if err != nil {
		return nil, nil, err
	}
	return affectedCategories, affectedTags, nil
}

func updatePostTranslation(ctx context.Context, tx *sql.Tx, existing int, input post.UpdateInput) error {
	if input.Title != nil {
		_, err := tx.ExecContext(ctx, `UPDATE post_i18n SET title = ? WHERE id = ?`, *input.Title, existing)
		if err != nil {
			return err
		}
	}
	if input.Slug != nil {
		_, err := tx.ExecContext(ctx, `UPDATE post_i18n SET slug = ? WHERE id = ?`, *input.Slug, existing)
		if err != nil {
			return err
		}
	}
	if input.Content != nil {
		_, err := tx.ExecContext(ctx, `UPDATE post_i18n SET content = ? WHERE id = ?`, *input.Content, existing)
		if err != nil {
			return err
		}
	}
	if input.Excerpt != nil {
		_, err := tx.ExecContext(ctx, `UPDATE post_i18n SET excerpt = ? WHERE id = ?`, *input.Excerpt, existing)
		if err != nil {
			return err
		}
	}
	if input.MetaTitle != nil {
		_, err := tx.ExecContext(ctx, `UPDATE post_i18n SET meta_title = ? WHERE id = ?`, *input.MetaTitle, existing)
		if err != nil {
			return err
		}
	}
	if input.MetaDescription != nil {
		_, err := tx.ExecContext(ctx, `UPDATE post_i18n SET meta_description = ? WHERE id = ?`, *input.MetaDescription, existing)
		if err != nil {
			return err
		}
	}
	return nil
}

// ApplyStatus sets the post status and stamps published_at the first time it is published.
func (r *PostRepository) ApplyStatus(ctx context.Context, id, status string) error {
	return applyStatus(ctx, r.db, id, status)
}

func applyStatus(ctx context.Context, ex execer, id, status string) error {
	var published sql.NullTime
	var err error
	err = ex.QueryRowContext(ctx, `SELECT published_at FROM posts WHERE id = ?`, id).Scan(&published)
	if err != nil {
		return err
	}
	if status == "published" && !published.Valid {
		_, err = ex.ExecContext(ctx, `UPDATE posts SET status = ?, published_at = UTC_TIMESTAMP() WHERE id = ?`, status, id)
		return err
	}
	_, err = ex.ExecContext(ctx, `UPDATE posts SET status = ? WHERE id = ?`, status, id)
	return err
}

// IncrementView adds one to view_count.
func (r *PostRepository) IncrementView(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE posts SET view_count = COALESCE(view_count,0) + 1 WHERE id = ?`, id)
	return err
}

// Delete removes a post and its dependent rows and returns the linked taxonomy ids.
func (r *PostRepository) Delete(ctx context.Context, id string) ([]int, []int, error) {
	var categoryIDs []int
	var tagIDs []int
	var err error
	categoryIDs, err = queryIDs(ctx, r.db, `SELECT category_id FROM posts_categories WHERE post_id = ?`, id)
	if err != nil {
		return nil, nil, err
	}
	tagIDs, err = queryIDs(ctx, r.db, `SELECT tag_id FROM posts_tags WHERE post_id = ?`, id)
	if err != nil {
		return nil, nil, err
	}
	var tx *sql.Tx
	tx, err = r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()
	for _, query := range []string{
		`DELETE FROM post_i18n WHERE post_id = ?`,
		`DELETE FROM post_metadata WHERE post_id = ?`,
		`DELETE FROM posts_categories WHERE post_id = ?`,
		`DELETE FROM posts_tags WHERE post_id = ?`,
		`DELETE FROM posts WHERE id = ?`,
	} {
		_, err = tx.ExecContext(ctx, query, id)
		if err != nil {
			return nil, nil, err
		}
	}
	err = tx.Commit()
	if err != nil {
		return nil, nil, err
	}
	return categoryIDs, tagIDs, nil
}

// ListMetadata returns metadata for one post. A post with no rows yields an empty slice.
func (r *PostRepository) ListMetadata(ctx context.Context, id string) ([]entity.Metadata, error) {
	var grouped map[string][]entity.Metadata
	var err error
	grouped, err = r.loadMetadata(ctx, []any{id})
	if err != nil {
		return nil, err
	}
	if grouped[id] == nil {
		return []entity.Metadata{}, nil
	}
	return grouped[id], nil
}

// ReplaceMetadata deletes and reinserts metadata, skipping blank keys.
func (r *PostRepository) ReplaceMetadata(ctx context.Context, id string, entries []entity.MetadataInput) error {
	var tx *sql.Tx
	var err error
	tx, err = r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `DELETE FROM post_metadata WHERE post_id = ?`, id)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if strings.TrimSpace(entry.MetaKey) == "" {
			continue
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO post_metadata (id, post_id, meta_key, meta_value) VALUES (?, ?, ?, ?)`,
			identity.NewUUID(), id, entry.MetaKey, entry.MetaValue)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func scanPost(sc interface{ Scan(...any) error }) (entity.PostRecord, error) {
	var row model.PostRow
	var err error
	err = sc.Scan(&row.ID, &row.AuthorID, &row.ContentTypeID, &row.FeaturedImageURL, &row.AccessLevel, &row.Status, &row.ViewCount, &row.PublishedAt, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		return entity.PostRecord{}, err
	}
	return entity.PostRecord{
		ID:               row.ID,
		AuthorID:         row.AuthorID,
		ContentTypeID:    row.ContentTypeID,
		FeaturedImageURL: StrPtr(row.FeaturedImageURL),
		AccessLevel:      row.AccessLevel,
		Status:           row.Status,
		ViewCount:        row.ViewCount,
		PublishedAt:      LocalFrom(row.PublishedAt),
		CreatedAt:        LocalFrom(row.CreatedAt),
		UpdatedAt:        LocalFrom(row.UpdatedAt),
	}, nil
}

func (r *PostRepository) loadI18n(ctx context.Context, ids []any) (map[string][]entity.PostTranslation, error) {
	out := map[string][]entity.PostTranslation{}
	query := `SELECT post_id, language_code, title, slug, content, excerpt, meta_title, meta_description
		FROM post_i18n WHERE post_id IN (` + placeholders(len(ids)) + `) ORDER BY id`
	var rows *sql.Rows
	var err error
	rows, err = r.db.QueryContext(ctx, query, ids...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var row model.PostI18nRow
		err = rows.Scan(&row.PostID, &row.LanguageCode, &row.Title, &row.Slug, &row.Content, &row.Excerpt, &row.MetaTitle, &row.MetaDescription)
		if err != nil {
			return nil, err
		}
		out[row.PostID] = append(out[row.PostID], entity.PostTranslation{
			LanguageCode:    row.LanguageCode,
			Title:           row.Title,
			Slug:            row.Slug,
			Content:         row.Content,
			Excerpt:         StrPtr(row.Excerpt),
			MetaTitle:       StrPtr(row.MetaTitle),
			MetaDescription: StrPtr(row.MetaDescription),
		})
	}
	return out, rows.Err()
}

func (r *PostRepository) loadTaxonomyLinks(ctx context.Context, ids []any) (map[string][]int, map[string][]int, error) {
	categories := map[string][]int{}
	tags := map[string][]int{}
	query := `SELECT post_id, category_id FROM posts_categories WHERE post_id IN (` + placeholders(len(ids)) + `)`
	var rows *sql.Rows
	var err error
	rows, err = r.db.QueryContext(ctx, query, ids...)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var id string
		var categoryID int
		err = rows.Scan(&id, &categoryID)
		if err != nil {
			rows.Close()
			return nil, nil, err
		}
		categories[id] = append(categories[id], categoryID)
	}
	rows.Close()
	query = `SELECT post_id, tag_id FROM posts_tags WHERE post_id IN (` + placeholders(len(ids)) + `)`
	rows, err = r.db.QueryContext(ctx, query, ids...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var tagID int
		err = rows.Scan(&id, &tagID)
		if err != nil {
			return nil, nil, err
		}
		tags[id] = append(tags[id], tagID)
	}
	return categories, tags, rows.Err()
}

func (r *PostRepository) loadMetadata(ctx context.Context, ids []any) (map[string][]entity.Metadata, error) {
	out := map[string][]entity.Metadata{}
	query := `SELECT id, post_id, meta_key, meta_value FROM post_metadata WHERE post_id IN (` + placeholders(len(ids)) + `)`
	var rows *sql.Rows
	var err error
	rows, err = r.db.QueryContext(ctx, query, ids...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var row model.MetadataRow
		err = rows.Scan(&row.ID, &row.PostID, &row.MetaKey, &row.MetaValue)
		if err != nil {
			return nil, err
		}
		out[row.PostID] = append(out[row.PostID], entity.Metadata{
			ID:        row.ID,
			PostID:    row.PostID,
			MetaKey:   row.MetaKey,
			MetaValue: StrPtr(row.MetaValue),
		})
	}
	return out, rows.Err()
}

func replaceLinks(ctx context.Context, tx *sql.Tx, postID string, categories, tags []int, doCategories, doTags bool) ([]int, []int, error) {
	var affectedCategories []int
	var affectedTags []int
	if doCategories {
		var previous []int
		var err error
		previous, err = idsFrom(ctx, tx, `SELECT category_id FROM posts_categories WHERE post_id = ?`, postID)
		if err != nil {
			return nil, nil, err
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM posts_categories WHERE post_id = ?`, postID)
		if err != nil {
			return nil, nil, err
		}
		next := map[int]struct{}{}
		for _, id := range categories {
			if _, ok := next[id]; ok {
				continue
			}
			next[id] = struct{}{}
			_, err = tx.ExecContext(ctx, `INSERT INTO posts_categories (post_id, category_id) VALUES (?, ?)`, postID, id)
			if err != nil {
				return nil, nil, err
			}
		}
		affectedCategories = unionIDs(previous, next)
	}
	if doTags {
		var previous []int
		var err error
		previous, err = idsFrom(ctx, tx, `SELECT tag_id FROM posts_tags WHERE post_id = ?`, postID)
		if err != nil {
			return nil, nil, err
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM posts_tags WHERE post_id = ?`, postID)
		if err != nil {
			return nil, nil, err
		}
		next := map[int]struct{}{}
		for _, id := range tags {
			if _, ok := next[id]; ok {
				continue
			}
			next[id] = struct{}{}
			_, err = tx.ExecContext(ctx, `INSERT INTO posts_tags (post_id, tag_id) VALUES (?, ?)`, postID, id)
			if err != nil {
				return nil, nil, err
			}
		}
		affectedTags = unionIDs(previous, next)
	}
	return affectedCategories, affectedTags, nil
}

func unionIDs(previous []int, next map[int]struct{}) []int {
	seen := map[int]struct{}{}
	out := make([]int, 0, len(previous)+len(next))
	for _, id := range previous {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for id := range next {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func idsFrom(ctx context.Context, tx *sql.Tx, query, id string) ([]int, error) {
	var rows *sql.Rows
	var err error
	rows, err = tx.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int
	for rows.Next() {
		var value int
		err = rows.Scan(&value)
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, rows.Err()
}

func queryIDs(ctx context.Context, db *sql.DB, query, id string) ([]int, error) {
	var rows *sql.Rows
	var err error
	rows, err = db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int
	for rows.Next() {
		var value int
		err = rows.Scan(&value)
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, rows.Err()
}
