package content

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"react-go-cms-content-service/internal/platform"
)

var nonWord = regexp.MustCompile(`[^\w\s-]`)
var spaces = regexp.MustCompile(`[\s_-]+`)
var edgeDash = regexp.MustCompile(`^-+|-+$`)

func normalizeLang(lang string) string {
	lang = strings.TrimSpace(strings.ToLower(lang))
	if lang == "" {
		return "en"
	}
	return lang
}

func slugify(text string) string {
	s := strings.ToLower(strings.TrimSpace(text))
	s = nonWord.ReplaceAllString(s, "")
	s = spaces.ReplaceAllString(s, "-")
	return edgeDash.ReplaceAllString(s, "")
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	b := strings.Repeat("?,", n)
	return b[:len(b)-1]
}

func isNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

type Page[T any] struct {
	Items []T   `json:"items"`
	Page  int   `json:"page"`
	Size  int   `json:"size"`
	Total int64 `json:"total"`
}

type Metadata struct {
	ID        string  `json:"id"`
	PostID    string  `json:"postId"`
	MetaKey   string  `json:"metaKey"`
	MetaValue *string `json:"metaValue"`
}

type LocalizedPost struct {
	ID               string             `json:"id"`
	AuthorID         string             `json:"authorId"`
	ContentTypeID    int                `json:"contentTypeId"`
	FeaturedImageURL *string            `json:"featuredImageUrl"`
	AccessLevel      string             `json:"accessLevel"`
	Status           string             `json:"status"`
	ViewCount        int                `json:"viewCount"`
	PublishedAt      platform.LocalJSON `json:"publishedAt"`
	CreatedAt        platform.LocalJSON `json:"createdAt"`
	UpdatedAt        platform.LocalJSON `json:"updatedAt"`
	CategoryIDs      []int              `json:"categoryIds"`
	TagIDs           []int              `json:"tagIds"`
	LanguageCode     string             `json:"languageCode"`
	Title            string             `json:"title"`
	Slug             string             `json:"slug"`
	Content          string             `json:"content"`
	Excerpt          *string            `json:"excerpt"`
	MetaTitle        *string            `json:"metaTitle"`
	MetaDescription  *string            `json:"metaDescription"`
	Metadata         []Metadata         `json:"metadata"`
}

type i18nRow struct {
	postID, lang, title, slug, content string
	excerpt, metaTitle, metaDesc       sql.NullString
}

func emptyStr() *string {
	s := ""
	return &s
}

func (s *Store) localizePosts(ctx context.Context, posts []postRow, lang string, withMeta bool) ([]LocalizedPost, error) {
	out := make([]LocalizedPost, 0, len(posts))
	if len(posts) == 0 {
		return out, nil
	}
	ids := make([]any, len(posts))
	for i, p := range posts {
		ids[i] = p.id
	}
	i18n, err := s.loadI18n(ctx, ids)
	if err != nil {
		return nil, err
	}
	cats, tags, err := s.loadTaxonomyLinks(ctx, ids)
	if err != nil {
		return nil, err
	}
	var meta map[string][]Metadata
	if withMeta {
		meta, err = s.loadMetadata(ctx, ids)
		if err != nil {
			return nil, err
		}
	}
	for _, p := range posts {
		dto := LocalizedPost{
			ID:               p.id,
			AuthorID:         p.authorID,
			ContentTypeID:    p.typeID,
			FeaturedImageURL: platform.StrPtr(p.image),
			AccessLevel:      p.access,
			Status:           p.status,
			ViewCount:        p.views,
			PublishedAt:      platform.LocalFrom(p.published),
			CreatedAt:        platform.LocalFrom(p.created),
			UpdatedAt:        platform.LocalFrom(p.updated),
			CategoryIDs:      cats[p.id],
			TagIDs:           tags[p.id],
			Metadata:         []Metadata{},
		}
		if dto.CategoryIDs == nil {
			dto.CategoryIDs = []int{}
		}
		if dto.TagIDs == nil {
			dto.TagIDs = []int{}
		}
		row := pickI18n(i18n[p.id], lang)
		if row == nil {
			dto.LanguageCode = lang
			dto.Title = ""
			dto.Slug = ""
			dto.Content = ""
			dto.Excerpt = emptyStr()
			dto.MetaTitle = emptyStr()
			dto.MetaDescription = emptyStr()
		} else {
			dto.LanguageCode = row.lang
			dto.Title = row.title
			dto.Slug = row.slug
			dto.Content = row.content
			dto.Excerpt = platform.StrPtr(row.excerpt)
			dto.MetaTitle = platform.StrPtr(row.metaTitle)
			dto.MetaDescription = platform.StrPtr(row.metaDesc)
		}
		if withMeta {
			if m := meta[p.id]; m != nil {
				dto.Metadata = m
			}
		}
		out = append(out, dto)
	}
	return out, nil
}

func pickI18n(rows []i18nRow, lang string) *i18nRow {
	if len(rows) == 0 {
		return nil
	}
	for i := range rows {
		if rows[i].lang == lang {
			return &rows[i]
		}
	}
	return &rows[0]
}

type postRow struct {
	id, authorID, access, status string
	typeID, views                int
	image                        sql.NullString
	published, created, updated  sql.NullTime
}

func scanPost(sc interface{ Scan(...any) error }) (postRow, error) {
	var p postRow
	err := sc.Scan(&p.id, &p.authorID, &p.typeID, &p.image, &p.access, &p.status, &p.views, &p.published, &p.created, &p.updated)
	return p, err
}

const postCols = `p.id, p.author_id, p.content_type_id, p.featured_image_url, p.access_level, p.status, COALESCE(p.view_count,0), p.published_at, p.created_at, p.updated_at`

func (s *Store) loadI18n(ctx context.Context, ids []any) (map[string][]i18nRow, error) {
	out := map[string][]i18nRow{}
	q := `SELECT post_id, language_code, title, slug, content, excerpt, meta_title, meta_description
		FROM post_i18n WHERE post_id IN (` + placeholders(len(ids)) + `) ORDER BY id`
	rows, err := s.db.QueryContext(ctx, q, ids...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var r i18nRow
		if err := rows.Scan(&r.postID, &r.lang, &r.title, &r.slug, &r.content, &r.excerpt, &r.metaTitle, &r.metaDesc); err != nil {
			return nil, err
		}
		out[r.postID] = append(out[r.postID], r)
	}
	return out, rows.Err()
}

func (s *Store) loadTaxonomyLinks(ctx context.Context, ids []any) (map[string][]int, map[string][]int, error) {
	cats := map[string][]int{}
	tags := map[string][]int{}
	q1 := `SELECT post_id, category_id FROM posts_categories WHERE post_id IN (` + placeholders(len(ids)) + `)`
	rows, err := s.db.QueryContext(ctx, q1, ids...)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var id string
		var cid int
		if err := rows.Scan(&id, &cid); err != nil {
			rows.Close()
			return nil, nil, err
		}
		cats[id] = append(cats[id], cid)
	}
	rows.Close()
	q2 := `SELECT post_id, tag_id FROM posts_tags WHERE post_id IN (` + placeholders(len(ids)) + `)`
	rows, err = s.db.QueryContext(ctx, q2, ids...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var tid int
		if err := rows.Scan(&id, &tid); err != nil {
			return nil, nil, err
		}
		tags[id] = append(tags[id], tid)
	}
	return cats, tags, rows.Err()
}

func (s *Store) loadMetadata(ctx context.Context, ids []any) (map[string][]Metadata, error) {
	out := map[string][]Metadata{}
	q := `SELECT id, post_id, meta_key, meta_value FROM post_metadata WHERE post_id IN (` + placeholders(len(ids)) + `)`
	rows, err := s.db.QueryContext(ctx, q, ids...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var m Metadata
		var val sql.NullString
		if err := rows.Scan(&m.ID, &m.PostID, &m.MetaKey, &val); err != nil {
			return nil, err
		}
		m.MetaValue = platform.StrPtr(val)
		out[m.PostID] = append(out[m.PostID], m)
	}
	return out, rows.Err()
}

type Store struct {
	db       *sql.DB
	authURL  string
	client   *http.Client
	popCache *popularCache
}

func NewStore(db *sql.DB, authURL string) *Store {
	return &Store{
		db:       db,
		authURL:  authURL,
		client:   &http.Client{Timeout: 5 * time.Second},
		popCache: newPopularCache(),
	}
}

func bad(msg string) error     { return platform.BadRequest("error", msg) }
func missing(msg string) error { return platform.NotFound("error", msg) }

func queryEscapeIDs(ids []string) string {
	return url.QueryEscape(strings.Join(ids, ","))
}
