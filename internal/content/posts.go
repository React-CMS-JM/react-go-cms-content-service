package content

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"react-go-cms-content-service/internal/platform"
)

var ownedSlugs = map[string]struct{}{
	"post": {}, "page": {}, "service": {}, "product": {},
}

func (s *Store) ListPosts(ctx context.Context, typ, status, lang string, page, size int) (Page[LocalizedPost], error) {
	language := normalizeLang(lang)
	if page < 0 {
		page = 0
	}
	if size < 1 {
		size = 1
	}
	if size > 100 {
		size = 100
	}
	where := ` FROM posts p JOIN content_types ct ON p.content_type_id = ct.id WHERE ct.slug <> 'course'`
	args := []any{}
	if strings.TrimSpace(typ) != "" {
		slug := strings.ToLower(strings.TrimSpace(typ))
		if slug == "course" {
			return Page[LocalizedPost]{}, bad("Unsupported content type: " + slug)
		}
		if _, ok := ownedSlugs[slug]; !ok {
			return Page[LocalizedPost]{}, bad("Unsupported content type: " + slug)
		}
		where += ` AND ct.slug = ?`
		args = append(args, slug)
	} else {
		where += ` AND ct.slug IN ('post','page','service','product')`
	}
	if strings.TrimSpace(status) != "" {
		where += ` AND p.status = ?`
		args = append(args, strings.TrimSpace(status))
	}
	var total int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*)`+where, args...).Scan(&total); err != nil {
		return Page[LocalizedPost]{}, err
	}
	q := `SELECT ` + postCols + where + ` ORDER BY p.created_at DESC LIMIT ? OFFSET ?`
	args = append(args, size, page*size)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return Page[LocalizedPost]{}, err
	}
	defer rows.Close()
	posts := []postRow{}
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return Page[LocalizedPost]{}, err
		}
		posts = append(posts, p)
	}
	items, err := s.localizePosts(ctx, posts, language, true)
	if err != nil {
		return Page[LocalizedPost]{}, err
	}
	return Page[LocalizedPost]{Items: items, Page: page, Size: size, Total: total}, nil
}

func (s *Store) GetPost(ctx context.Context, id, lang string) (LocalizedPost, error) {
	p, err := s.requireOwned(ctx, id)
	if err != nil {
		return LocalizedPost{}, err
	}
	items, err := s.localizePosts(ctx, []postRow{p}, normalizeLang(lang), true)
	if err != nil || len(items) == 0 {
		return LocalizedPost{}, err
	}
	return items[0], nil
}

func (s *Store) GetPostBySlug(ctx context.Context, slug, typ, lang string) (LocalizedPost, error) {
	language := normalizeLang(lang)
	var postID string
	err := s.db.QueryRowContext(ctx, `SELECT post_id FROM post_i18n WHERE slug = ? AND language_code = ? ORDER BY id LIMIT 1`, slug, language).Scan(&postID)
	if isNotFound(err) {
		err = s.db.QueryRowContext(ctx, `SELECT post_id FROM post_i18n WHERE slug = ? ORDER BY id LIMIT 1`, slug).Scan(&postID)
	}
	if isNotFound(err) {
		return LocalizedPost{}, missing("Post not found for slug: " + slug)
	}
	if err != nil {
		return LocalizedPost{}, err
	}
	p, err := s.requireOwned(ctx, postID)
	if err != nil {
		return LocalizedPost{}, err
	}
	if strings.TrimSpace(typ) != "" {
		var typeSlug string
		if err := s.db.QueryRowContext(ctx, `SELECT slug FROM content_types WHERE id = ?`, p.typeID).Scan(&typeSlug); err != nil || !strings.EqualFold(typeSlug, typ) {
			return LocalizedPost{}, missing("Post not found for slug/type: " + slug + "/" + typ)
		}
	}
	items, err := s.localizePosts(ctx, []postRow{p}, language, true)
	if err != nil || len(items) == 0 {
		return LocalizedPost{}, err
	}
	return items[0], nil
}

type CreatePost struct {
	AuthorID         *string `json:"authorId"`
	ContentTypeID    *int    `json:"contentTypeId"`
	ContentTypeSlug  *string `json:"contentTypeSlug"`
	FeaturedImageURL *string `json:"featuredImageUrl"`
	AccessLevel      *string `json:"accessLevel"`
	Status           *string `json:"status"`
	CategoryIDs      []int   `json:"categoryIds"`
	TagIDs           []int   `json:"tagIds"`
	LanguageCode     *string `json:"languageCode"`
	Title            string  `json:"title"`
	Slug             *string `json:"slug"`
	Content          *string `json:"content"`
	Excerpt          *string `json:"excerpt"`
	MetaTitle        *string `json:"metaTitle"`
	MetaDescription  *string `json:"metaDescription"`
}

func (s *Store) CreatePost(ctx context.Context, req CreatePost, fallbackAuthor string) (LocalizedPost, error) {
	if strings.TrimSpace(req.Title) == "" {
		return LocalizedPost{}, bad("title is required")
	}
	if req.Content == nil {
		return LocalizedPost{}, bad("content is required")
	}
	typeID, err := s.resolveType(ctx, req.ContentTypeID, req.ContentTypeSlug)
	if err != nil {
		return LocalizedPost{}, err
	}
	author := fallbackAuthor
	if req.AuthorID != nil && strings.TrimSpace(*req.AuthorID) != "" {
		author = strings.TrimSpace(*req.AuthorID)
	}
	if strings.TrimSpace(author) == "" {
		return LocalizedPost{}, bad("authorId is required")
	}
	lang := "en"
	if req.LanguageCode != nil {
		lang = normalizeLang(*req.LanguageCode)
	}
	slug := slugify(req.Title)
	if req.Slug != nil && strings.TrimSpace(*req.Slug) != "" {
		slug = strings.TrimSpace(*req.Slug)
	}
	access := "public"
	if req.AccessLevel != nil {
		access = *req.AccessLevel
	}
	status := "draft"
	if req.Status != nil {
		status = *req.Status
	}
	id := platform.NewUUID()
	var published any
	if status == "published" {
		published = time.Now().UTC().Format("2006-01-02 15:04:05")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return LocalizedPost{}, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO posts (id, author_id, content_type_id, featured_image_url, access_level, status, view_count, published_at)
		VALUES (?, ?, ?, ?, ?, ?, 0, ?)`,
		id, author, typeID, req.FeaturedImageURL, access, status, published); err != nil {
		return LocalizedPost{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO post_i18n (post_id, language_code, title, slug, content, excerpt, meta_title, meta_description)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, lang, req.Title, slug, *req.Content, req.Excerpt, req.MetaTitle, req.MetaDescription); err != nil {
		return LocalizedPost{}, err
	}
	affectedCats, affectedTags, err := replaceLinks(ctx, tx, id, req.CategoryIDs, req.TagIDs, true, true)
	if err != nil {
		return LocalizedPost{}, err
	}
	if err := tx.Commit(); err != nil {
		return LocalizedPost{}, err
	}
	_ = s.refreshCategoryCounts(ctx, affectedCats)
	_ = s.refreshTagCounts(ctx, affectedTags)
	return s.GetPost(ctx, id, lang)
}

type UpdatePost struct {
	ContentTypeID    *int    `json:"contentTypeId"`
	FeaturedImageURL *string `json:"featuredImageUrl"`
	AccessLevel      *string `json:"accessLevel"`
	Status           *string `json:"status"`
	CategoryIDs      *[]int  `json:"categoryIds"`
	TagIDs           *[]int  `json:"tagIds"`
	LanguageCode     *string `json:"languageCode"`
	Title            *string `json:"title"`
	Slug             *string `json:"slug"`
	Content          *string `json:"content"`
	Excerpt          *string `json:"excerpt"`
	MetaTitle        *string `json:"metaTitle"`
	MetaDescription  *string `json:"metaDescription"`
}

func (s *Store) UpdatePost(ctx context.Context, id string, req UpdatePost) (LocalizedPost, error) {
	if _, err := s.requireOwned(ctx, id); err != nil {
		return LocalizedPost{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return LocalizedPost{}, err
	}
	defer tx.Rollback()
	if req.ContentTypeID != nil {
		typeID, err := s.resolveType(ctx, req.ContentTypeID, nil)
		if err != nil {
			return LocalizedPost{}, err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE posts SET content_type_id = ? WHERE id = ?`, typeID, id); err != nil {
			return LocalizedPost{}, err
		}
	}
	if req.FeaturedImageURL != nil {
		if _, err := tx.ExecContext(ctx, `UPDATE posts SET featured_image_url = ? WHERE id = ?`, *req.FeaturedImageURL, id); err != nil {
			return LocalizedPost{}, err
		}
	}
	if req.AccessLevel != nil {
		if _, err := tx.ExecContext(ctx, `UPDATE posts SET access_level = ? WHERE id = ?`, *req.AccessLevel, id); err != nil {
			return LocalizedPost{}, err
		}
	}
	if req.Status != nil {
		if err := applyStatus(ctx, tx, id, *req.Status); err != nil {
			return LocalizedPost{}, err
		}
	}
	var affectedCats, affectedTags []int
	if req.CategoryIDs != nil || req.TagIDs != nil {
		var cats, tags []int
		replaceCats, replaceTags := false, false
		if req.CategoryIDs != nil {
			cats = *req.CategoryIDs
			replaceCats = true
		}
		if req.TagIDs != nil {
			tags = *req.TagIDs
			replaceTags = true
		}
		affectedCats, affectedTags, err = replaceLinks(ctx, tx, id, cats, tags, replaceCats, replaceTags)
		if err != nil {
			return LocalizedPost{}, err
		}
	}
	lang := "en"
	if req.LanguageCode != nil {
		lang = normalizeLang(*req.LanguageCode)
	}
	hasI18n := req.Title != nil || req.Slug != nil || req.Content != nil || req.Excerpt != nil || req.MetaTitle != nil || req.MetaDescription != nil
	if hasI18n {
		var existing int
		err := tx.QueryRowContext(ctx, `SELECT id FROM post_i18n WHERE post_id = ? AND language_code = ?`, id, lang).Scan(&existing)
		if isNotFound(err) {
			title := ""
			if req.Title != nil {
				title = *req.Title
			}
			slug := slugify(title)
			if req.Slug != nil {
				slug = *req.Slug
			}
			content := ""
			if req.Content != nil {
				content = *req.Content
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO post_i18n (post_id, language_code, title, slug, content, excerpt, meta_title, meta_description)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				id, lang, title, slug, content, req.Excerpt, req.MetaTitle, req.MetaDescription); err != nil {
				return LocalizedPost{}, err
			}
		} else if err != nil {
			return LocalizedPost{}, err
		} else {
			if req.Title != nil {
				if _, err := tx.ExecContext(ctx, `UPDATE post_i18n SET title = ? WHERE id = ?`, *req.Title, existing); err != nil {
					return LocalizedPost{}, err
				}
			}
			if req.Slug != nil {
				if _, err := tx.ExecContext(ctx, `UPDATE post_i18n SET slug = ? WHERE id = ?`, *req.Slug, existing); err != nil {
					return LocalizedPost{}, err
				}
			}
			if req.Content != nil {
				if _, err := tx.ExecContext(ctx, `UPDATE post_i18n SET content = ? WHERE id = ?`, *req.Content, existing); err != nil {
					return LocalizedPost{}, err
				}
			}
			if req.Excerpt != nil {
				if _, err := tx.ExecContext(ctx, `UPDATE post_i18n SET excerpt = ? WHERE id = ?`, *req.Excerpt, existing); err != nil {
					return LocalizedPost{}, err
				}
			}
			if req.MetaTitle != nil {
				if _, err := tx.ExecContext(ctx, `UPDATE post_i18n SET meta_title = ? WHERE id = ?`, *req.MetaTitle, existing); err != nil {
					return LocalizedPost{}, err
				}
			}
			if req.MetaDescription != nil {
				if _, err := tx.ExecContext(ctx, `UPDATE post_i18n SET meta_description = ? WHERE id = ?`, *req.MetaDescription, existing); err != nil {
					return LocalizedPost{}, err
				}
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return LocalizedPost{}, err
	}
	_ = s.refreshCategoryCounts(ctx, affectedCats)
	_ = s.refreshTagCounts(ctx, affectedTags)
	return s.GetPost(ctx, id, lang)
}

func (s *Store) PatchPostStatus(ctx context.Context, id, status string) (LocalizedPost, error) {
	if strings.TrimSpace(status) == "" {
		return LocalizedPost{}, bad("status is required")
	}
	if _, err := s.requireOwned(ctx, id); err != nil {
		return LocalizedPost{}, err
	}
	if err := applyStatus(ctx, s.db, id, strings.TrimSpace(status)); err != nil {
		return LocalizedPost{}, err
	}
	return s.GetPost(ctx, id, "en")
}

func (s *Store) IncrementView(ctx context.Context, id string) (LocalizedPost, error) {
	if _, err := s.requireOwned(ctx, id); err != nil {
		return LocalizedPost{}, err
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE posts SET view_count = COALESCE(view_count,0) + 1 WHERE id = ?`, id); err != nil {
		return LocalizedPost{}, err
	}
	return s.GetPost(ctx, id, "en")
}

func (s *Store) DeletePost(ctx context.Context, id string) error {
	if _, err := s.requireOwned(ctx, id); err != nil {
		return err
	}
	catIDs, tagIDs, err := s.linkIDs(ctx, id)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, q := range []string{
		`DELETE FROM post_i18n WHERE post_id = ?`,
		`DELETE FROM post_metadata WHERE post_id = ?`,
		`DELETE FROM posts_categories WHERE post_id = ?`,
		`DELETE FROM posts_tags WHERE post_id = ?`,
		`DELETE FROM posts WHERE id = ?`,
	} {
		if _, err := tx.ExecContext(ctx, q, id); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	_ = s.refreshCategoryCounts(ctx, catIDs)
	_ = s.refreshTagCounts(ctx, tagIDs)
	return nil
}

func (s *Store) GetMetadata(ctx context.Context, id string) ([]Metadata, error) {
	if _, err := s.requireOwned(ctx, id); err != nil {
		return nil, err
	}
	m, err := s.loadMetadata(ctx, []any{id})
	if err != nil {
		return nil, err
	}
	if m[id] == nil {
		return []Metadata{}, nil
	}
	return m[id], nil
}

type MetaEntry struct {
	MetaKey   string  `json:"metaKey"`
	MetaValue *string `json:"metaValue"`
}

func (s *Store) PutMetadata(ctx context.Context, id string, entries []MetaEntry) ([]Metadata, error) {
	if _, err := s.requireOwned(ctx, id); err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM post_metadata WHERE post_id = ?`, id); err != nil {
		return nil, err
	}
	for _, e := range entries {
		if strings.TrimSpace(e.MetaKey) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO post_metadata (id, post_id, meta_key, meta_value) VALUES (?, ?, ?, ?)`,
			platform.NewUUID(), id, e.MetaKey, e.MetaValue); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetMetadata(ctx, id)
}

func (s *Store) requireOwned(ctx context.Context, id string) (postRow, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+postCols+` FROM posts p WHERE p.id = ?`, id)
	p, err := scanPost(row)
	if isNotFound(err) {
		return postRow{}, missing("Post not found: " + id)
	}
	if err != nil {
		return postRow{}, err
	}
	var slug string
	if err := s.db.QueryRowContext(ctx, `SELECT slug FROM content_types WHERE id = ?`, p.typeID).Scan(&slug); err != nil {
		return postRow{}, missing("Post not found: " + id)
	}
	if slug == "course" {
		return postRow{}, missing("Post not found: " + id)
	}
	if _, ok := ownedSlugs[slug]; !ok {
		return postRow{}, missing("Post not found: " + id)
	}
	return p, nil
}

func (s *Store) resolveType(ctx context.Context, id *int, slug *string) (int, error) {
	var typeID int
	var typeSlug string
	var err error
	if id != nil {
		err = s.db.QueryRowContext(ctx, `SELECT id, slug FROM content_types WHERE id = ?`, *id).Scan(&typeID, &typeSlug)
	} else if slug != nil && strings.TrimSpace(*slug) != "" {
		err = s.db.QueryRowContext(ctx, `SELECT id, slug FROM content_types WHERE slug = ?`, strings.ToLower(strings.TrimSpace(*slug))).Scan(&typeID, &typeSlug)
	} else {
		return 0, bad("contentTypeId or contentTypeSlug is required")
	}
	if isNotFound(err) {
		return 0, bad("contentTypeId or contentTypeSlug is required")
	}
	if err != nil {
		return 0, err
	}
	if typeSlug == "course" {
		return 0, bad("Content type not managed by content-service: " + typeSlug)
	}
	if _, ok := ownedSlugs[typeSlug]; !ok {
		return 0, bad("Content type not managed by content-service: " + typeSlug)
	}
	return typeID, nil
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func applyStatus(ctx context.Context, ex execer, id, status string) error {
	var published sql.NullTime
	if err := ex.QueryRowContext(ctx, `SELECT published_at FROM posts WHERE id = ?`, id).Scan(&published); err != nil {
		return err
	}
	if status == "published" && !published.Valid {
		_, err := ex.ExecContext(ctx, `UPDATE posts SET status = ?, published_at = UTC_TIMESTAMP() WHERE id = ?`, status, id)
		return err
	}
	_, err := ex.ExecContext(ctx, `UPDATE posts SET status = ? WHERE id = ?`, status, id)
	return err
}

func replaceLinks(ctx context.Context, tx *sql.Tx, postID string, cats, tags []int, doCats, doTags bool) ([]int, []int, error) {
	var affectedCats, affectedTags []int
	if doCats {
		prev, err := idsFrom(ctx, tx, `SELECT category_id FROM posts_categories WHERE post_id = ?`, postID)
		if err != nil {
			return nil, nil, err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM posts_categories WHERE post_id = ?`, postID); err != nil {
			return nil, nil, err
		}
		next := map[int]struct{}{}
		for _, id := range cats {
			if _, ok := next[id]; ok {
				continue
			}
			next[id] = struct{}{}
			if _, err := tx.ExecContext(ctx, `INSERT INTO posts_categories (post_id, category_id) VALUES (?, ?)`, postID, id); err != nil {
				return nil, nil, err
			}
		}
		affectedCats = unionIDs(prev, next)
	}
	if doTags {
		prev, err := idsFrom(ctx, tx, `SELECT tag_id FROM posts_tags WHERE post_id = ?`, postID)
		if err != nil {
			return nil, nil, err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM posts_tags WHERE post_id = ?`, postID); err != nil {
			return nil, nil, err
		}
		next := map[int]struct{}{}
		for _, id := range tags {
			if _, ok := next[id]; ok {
				continue
			}
			next[id] = struct{}{}
			if _, err := tx.ExecContext(ctx, `INSERT INTO posts_tags (post_id, tag_id) VALUES (?, ?)`, postID, id); err != nil {
				return nil, nil, err
			}
		}
		affectedTags = unionIDs(prev, next)
	}
	return affectedCats, affectedTags, nil
}

func unionIDs(prev []int, next map[int]struct{}) []int {
	seen := map[int]struct{}{}
	out := make([]int, 0, len(prev)+len(next))
	for _, id := range prev {
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

func idsFrom(ctx context.Context, tx *sql.Tx, q, id string) ([]int, error) {
	rows, err := tx.QueryContext(ctx, q, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int
	for rows.Next() {
		var n int
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}
