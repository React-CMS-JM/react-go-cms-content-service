package content

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"react-go-cms-content-service/internal/platform"
)

type Dashboard struct {
	Counts          typeCounts `json:"counts"`
	PendingComments int64      `json:"pendingComments"`
	Recent          []Recent   `json:"recent"`
}

type typeCounts struct {
	Post, Page, Service, Product int64
}

func (c typeCounts) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, `{"post":%d,"page":%d,"service":%d,"product":%d}`, c.Post, c.Page, c.Service, c.Product), nil
}

type Recent struct {
	ID              string             `json:"id"`
	Title           string             `json:"title"`
	AuthorID        string             `json:"authorId"`
	AuthorName      string             `json:"authorName"`
	Status          string             `json:"status"`
	ContentTypeSlug string             `json:"contentTypeSlug"`
	UpdatedAt       platform.LocalJSON `json:"updatedAt"`
}

func (s *Store) Dashboard(ctx context.Context, lang string, recentLimit int, authorization string) (Dashboard, error) {
	language := normalizeLang(lang)
	if recentLimit <= 0 {
		recentLimit = 6
	}
	if recentLimit > 20 {
		recentLimit = 20
	}
	var d Dashboard
	d.Recent = []Recent{}
	d.Counts.Post = s.countSlug(ctx, "post")
	d.Counts.Page = s.countSlug(ctx, "page")
	d.Counts.Service = s.countSlug(ctx, "service")
	d.Counts.Product = s.countSlug(ctx, "product")
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM comments WHERE status = 'pending'`).Scan(&d.PendingComments)

	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.author_id, p.content_type_id, p.status, p.updated_at
		FROM posts p ORDER BY p.updated_at DESC LIMIT ?`, recentLimit)
	if err != nil {
		return d, err
	}
	defer rows.Close()
	type raw struct {
		id, author, status string
		typeID             int
		updated            sql.NullTime
	}
	var list []raw
	authors := []string{}
	for rows.Next() {
		var r raw
		if err := rows.Scan(&r.id, &r.author, &r.typeID, &r.status, &r.updated); err != nil {
			return d, err
		}
		list = append(list, r)
		if strings.TrimSpace(r.author) != "" {
			authors = append(authors, r.author)
		}
	}
	if err := rows.Err(); err != nil {
		return d, err
	}
	names := s.authorNames(ctx, authors, authorization)
	for _, r := range list {
		var slug string
		_ = s.db.QueryRowContext(ctx, `SELECT slug FROM content_types WHERE id = ?`, r.typeID).Scan(&slug)
		if slug == "" {
			slug = "post"
		}
		title := ""
		err := s.db.QueryRowContext(ctx, `SELECT title FROM post_i18n WHERE post_id = ? AND language_code = ? LIMIT 1`, r.id, language).Scan(&title)
		if isNotFound(err) {
			_ = s.db.QueryRowContext(ctx, `SELECT title FROM post_i18n WHERE post_id = ? ORDER BY id LIMIT 1`, r.id).Scan(&title)
		}
		d.Recent = append(d.Recent, Recent{
			ID:              r.id,
			Title:           title,
			AuthorID:        r.author,
			AuthorName:      names[r.author],
			Status:          r.status,
			ContentTypeSlug: slug,
			UpdatedAt:       platform.LocalFrom(r.updated),
		})
	}
	return d, nil
}

func (s *Store) countSlug(ctx context.Context, slug string) int64 {
	var n int64
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM posts p JOIN content_types ct ON ct.id = p.content_type_id WHERE ct.slug = ?`, slug).Scan(&n)
	if err != nil {
		return 0
	}
	return n
}

type userSummary struct {
	ID        string  `json:"id"`
	FirstName *string `json:"firstName"`
	LastName  *string `json:"lastName"`
}

func (s *Store) authorNames(ctx context.Context, ids []string, authorization string) map[string]string {
	out := map[string]string{}
	if len(ids) == 0 || strings.TrimSpace(authorization) == "" {
		return out
	}
	seen := map[string]struct{}{}
	uniq := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.authURL+"/api/users/by-ids?ids="+queryEscapeIDs(uniq), nil)
	if err != nil {
		return out
	}
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Accept", "application/json")
	res, err := s.client.Do(req)
	if err != nil {
		return out
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return out
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return out
	}
	var rows []userSummary
	if json.Unmarshal(body, &rows) != nil {
		return out
	}
	for _, u := range rows {
		first, last := "", ""
		if u.FirstName != nil {
			first = strings.TrimSpace(*u.FirstName)
		}
		if u.LastName != nil {
			last = strings.TrimSpace(*u.LastName)
		}
		out[u.ID] = strings.TrimSpace(first + " " + last)
	}
	return out
}
