package content

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"time"

	"react-go-cms-content-service/internal/platform"
)

type Taxonomy struct {
	ID            int     `json:"id"`
	DBDescription *string `json:"dbDescription"`
	LanguageCode  string  `json:"languageCode"`
	Name          string  `json:"name"`
	Slug          string  `json:"slug"`
	UsageCount    int64   `json:"usageCount"`
}

type TaxonomyIn struct {
	LanguageCode  *string `json:"languageCode"`
	Name          *string `json:"name"`
	Slug          *string `json:"slug"`
	DBDescription *string `json:"dbDescription"`
}

type popularCache struct {
	mu    sync.Mutex
	items map[string]popularEntry
}

type popularEntry struct {
	at   time.Time
	cats []Taxonomy
	tags []Taxonomy
	kind string
}

func newPopularCache() *popularCache {
	return &popularCache{items: map[string]popularEntry{}}
}

func (c *popularCache) get(key string) (popularEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.items[key]
	if !ok || time.Since(e.at) > 5*time.Minute {
		return popularEntry{}, false
	}
	return e, true
}

func (c *popularCache) put(key string, e popularEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.items) >= 32 {
		c.items = map[string]popularEntry{}
	}
	e.at = time.Now()
	c.items[key] = e
}

func (c *popularCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = map[string]popularEntry{}
}

func (s *Store) ListPopularCategories(ctx context.Context, lang string) ([]Taxonomy, error) {
	language := normalizeLang(lang)
	key := "cat:" + language
	if e, ok := s.popCache.get(key); ok {
		return e.cats, nil
	}
	rows, err := s.listTaxonomy(ctx, "categories", "category", language, 100, "")
	if err != nil {
		return nil, err
	}
	s.popCache.put(key, popularEntry{cats: rows, kind: "cat"})
	return rows, nil
}

func (s *Store) ListPopularTags(ctx context.Context, lang string) ([]Taxonomy, error) {
	language := normalizeLang(lang)
	key := "tag:" + language
	if e, ok := s.popCache.get(key); ok {
		return e.tags, nil
	}
	rows, err := s.listTaxonomy(ctx, "tags", "tag", language, 100, "")
	if err != nil {
		return nil, err
	}
	s.popCache.put(key, popularEntry{tags: rows, kind: "tag"})
	return rows, nil
}

func (s *Store) SearchCategories(ctx context.Context, q, lang string, limit int) ([]Taxonomy, error) {
	return s.searchTaxonomy(ctx, "categories", "category", q, lang, limit)
}

func (s *Store) SearchTags(ctx context.Context, q, lang string, limit int) ([]Taxonomy, error) {
	return s.searchTaxonomy(ctx, "tags", "tag", q, lang, limit)
}

func (s *Store) CategoriesByIDs(ctx context.Context, ids []int, lang string) ([]Taxonomy, error) {
	return s.taxonomyByIDs(ctx, "categories", "category", ids, lang)
}

func (s *Store) TagsByIDs(ctx context.Context, ids []int, lang string) ([]Taxonomy, error) {
	return s.taxonomyByIDs(ctx, "tags", "tag", ids, lang)
}

func (s *Store) AdminCategories(ctx context.Context, lang string, page, size int, q string) (Page[Taxonomy], error) {
	return s.adminTaxonomy(ctx, "categories", "category", lang, page, size, q)
}

func (s *Store) AdminTags(ctx context.Context, lang string, page, size int, q string) (Page[Taxonomy], error) {
	return s.adminTaxonomy(ctx, "tags", "tag", lang, page, size, q)
}

func (s *Store) CreateCategory(ctx context.Context, req TaxonomyIn) (Taxonomy, error) {
	return s.createTaxonomy(ctx, "categories", "category", req)
}

func (s *Store) CreateTag(ctx context.Context, req TaxonomyIn) (Taxonomy, error) {
	return s.createTaxonomy(ctx, "tags", "tag", req)
}

func (s *Store) UpdateCategory(ctx context.Context, id int, req TaxonomyIn) (Taxonomy, error) {
	return s.updateTaxonomy(ctx, "categories", "category", id, req)
}

func (s *Store) UpdateTag(ctx context.Context, id int, req TaxonomyIn) (Taxonomy, error) {
	return s.updateTaxonomy(ctx, "tags", "tag", id, req)
}

func (s *Store) DeleteCategory(ctx context.Context, id int) error {
	return s.deleteTaxonomy(ctx, "categories", "category", "posts_categories", "category_id", id, "Category")
}

func (s *Store) DeleteTag(ctx context.Context, id int) error {
	return s.deleteTaxonomy(ctx, "tags", "tag", "posts_tags", "tag_id", id, "Tag")
}

func (s *Store) listTaxonomy(ctx context.Context, table, kind, lang string, limit int, extraWhere string, args ...any) ([]Taxonomy, error) {
	q := `SELECT id, db_description, usage_count FROM ` + table + extraWhere +
		` ORDER BY usage_count DESC, updated_at DESC, created_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	base := []taxBase{}
	for rows.Next() {
		var b taxBase
		var desc sql.NullString
		if err := rows.Scan(&b.id, &desc, &b.usage); err != nil {
			return nil, err
		}
		b.desc = desc
		base = append(base, b)
	}
	return s.hydrateTaxonomy(ctx, kind, lang, base)
}

type taxBase struct {
	id    int
	desc  sql.NullString
	usage int64
}

func (s *Store) hydrateTaxonomy(ctx context.Context, kind, lang string, base []taxBase) ([]Taxonomy, error) {
	out := make([]Taxonomy, 0, len(base))
	if len(base) == 0 {
		return out, nil
	}
	ids := make([]any, len(base))
	for i, b := range base {
		ids[i] = b.id
	}
	col := kind + "_id"
	q := `SELECT ` + col + `, language_code, name, slug FROM ` + kind + `_i18n WHERE ` + col + ` IN (` + placeholders(len(ids)) + `) ORDER BY id`
	rows, err := s.db.QueryContext(ctx, q, ids...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type tr struct{ lang, name, slug string }
	byID := map[int][]tr{}
	for rows.Next() {
		var id int
		var t tr
		if err := rows.Scan(&id, &t.lang, &t.name, &t.slug); err != nil {
			return nil, err
		}
		byID[id] = append(byID[id], t)
	}
	for _, b := range base {
		item := Taxonomy{
			ID:            b.id,
			DBDescription: platform.StrPtr(b.desc),
			UsageCount:    b.usage,
			LanguageCode:  lang,
			Name:          "",
			Slug:          "",
		}
		rowsT := byID[b.id]
		chosen := -1
		for i, t := range rowsT {
			if t.lang == lang {
				chosen = i
				break
			}
		}
		if chosen < 0 && len(rowsT) > 0 {
			chosen = 0
		}
		if chosen >= 0 {
			item.LanguageCode = rowsT[chosen].lang
			item.Name = rowsT[chosen].name
			item.Slug = rowsT[chosen].slug
		}
		out = append(out, item)
	}
	return out, nil
}

func normalizeSearch(q string) string {
	q = strings.TrimSpace(q)
	compact := strings.ReplaceAll(q, " ", "")
	if len(compact) < 2 {
		return ""
	}
	return q
}

func (s *Store) searchTaxonomy(ctx context.Context, table, kind, q, lang string, limit int) ([]Taxonomy, error) {
	query := normalizeSearch(q)
	if query == "" {
		return []Taxonomy{}, nil
	}
	if limit <= 0 || limit > 20 {
		limit = 20
	}
	language := normalizeLang(lang)
	pattern := "%" + strings.ToLower(query) + "%"
	col := kind + "_id"
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT i.`+col+` FROM `+kind+`_i18n i
		WHERE LOWER(i.name) LIKE ? OR LOWER(i.slug) LIKE ?
		LIMIT ?`, pattern, pattern, limit*4)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return []Taxonomy{}, nil
	}
	items, err := s.taxonomyByIDs(ctx, table, kind, ids, language)
	if err != nil {
		return nil, err
	}
	byID := map[int]Taxonomy{}
	for _, it := range items {
		byID[it.ID] = it
	}
	ordered := make([]Taxonomy, 0, len(ids))
	for _, id := range ids {
		if it, ok := byID[id]; ok {
			ordered = append(ordered, it)
		}
	}
	sortTaxonomy(ordered)
	if len(ordered) > limit {
		ordered = ordered[:limit]
	}
	return ordered, nil
}

func sortTaxonomy(items []Taxonomy) {
	for i := 1; i < len(items); i++ {
		j := i
		for j > 0 && taxLess(items[j], items[j-1]) {
			items[j], items[j-1] = items[j-1], items[j]
			j--
		}
	}
}

func taxLess(a, b Taxonomy) bool {
	if a.UsageCount != b.UsageCount {
		return a.UsageCount > b.UsageCount
	}
	return a.ID > b.ID
}

func (s *Store) taxonomyByIDs(ctx context.Context, table, kind string, ids []int, lang string) ([]Taxonomy, error) {
	if len(ids) == 0 {
		return []Taxonomy{}, nil
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
		return []Taxonomy{}, nil
	}
	q := `SELECT id, db_description, usage_count FROM ` + table + ` WHERE id IN (` + placeholders(len(args)) + `)`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	base := []taxBase{}
	for rows.Next() {
		var b taxBase
		var desc sql.NullString
		if err := rows.Scan(&b.id, &desc, &b.usage); err != nil {
			return nil, err
		}
		b.desc = desc
		base = append(base, b)
	}
	return s.hydrateTaxonomy(ctx, kind, normalizeLang(lang), base)
}

func (s *Store) adminTaxonomy(ctx context.Context, table, kind, lang string, page, size int, q string) (Page[Taxonomy], error) {
	language := normalizeLang(lang)
	if page < 0 {
		page = 0
	}
	if size <= 0 {
		size = 10
	}
	if size > 50 {
		size = 50
	}
	query := normalizeSearch(q)
	var total int64
	var base []taxBase
	if query == "" {
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table).Scan(&total); err != nil {
			return Page[Taxonomy]{}, err
		}
		qsql := `SELECT id, db_description, usage_count FROM ` + table + ` ORDER BY usage_count DESC, updated_at DESC, created_at DESC LIMIT ? OFFSET ?`
		rws, err := s.db.QueryContext(ctx, qsql, size, page*size)
		if err != nil {
			return Page[Taxonomy]{}, err
		}
		defer rws.Close()
		for rws.Next() {
			var b taxBase
			var desc sql.NullString
			if err := rws.Scan(&b.id, &desc, &b.usage); err != nil {
				return Page[Taxonomy]{}, err
			}
			b.desc = desc
			base = append(base, b)
		}
	} else {
		pattern := "%" + strings.ToLower(query) + "%"
		col := kind + "_id"
		countSQL := `SELECT COUNT(DISTINCT t.id) FROM ` + table + ` t JOIN ` + kind + `_i18n i ON i.` + col + ` = t.id WHERE LOWER(i.name) LIKE ? OR LOWER(i.slug) LIKE ?`
		if err := s.db.QueryRowContext(ctx, countSQL, pattern, pattern).Scan(&total); err != nil {
			return Page[Taxonomy]{}, err
		}
		idSQL := `SELECT t.id FROM ` + table + ` t JOIN ` + kind + `_i18n i ON i.` + col + ` = t.id
			WHERE LOWER(i.name) LIKE ? OR LOWER(i.slug) LIKE ?
			GROUP BY t.id, t.usage_count, t.updated_at, t.created_at
			ORDER BY t.usage_count DESC, t.updated_at DESC, t.created_at DESC
			LIMIT ? OFFSET ?`
		rws, err := s.db.QueryContext(ctx, idSQL, pattern, pattern, size, page*size)
		if err != nil {
			return Page[Taxonomy]{}, err
		}
		defer rws.Close()
		var ids []int
		for rws.Next() {
			var id int
			if err := rws.Scan(&id); err != nil {
				return Page[Taxonomy]{}, err
			}
			ids = append(ids, id)
		}
		items, err := s.taxonomyByIDs(ctx, table, kind, ids, language)
		if err != nil {
			return Page[Taxonomy]{}, err
		}
		by := map[int]Taxonomy{}
		for _, it := range items {
			by[it.ID] = it
		}
		ordered := make([]Taxonomy, 0, len(ids))
		for _, id := range ids {
			if it, ok := by[id]; ok {
				ordered = append(ordered, it)
			}
		}
		if ordered == nil {
			ordered = []Taxonomy{}
		}
		return Page[Taxonomy]{Items: ordered, Page: page, Size: size, Total: total}, nil
	}
	items, err := s.hydrateTaxonomy(ctx, kind, language, base)
	if err != nil {
		return Page[Taxonomy]{}, err
	}
	return Page[Taxonomy]{Items: items, Page: page, Size: size, Total: total}, nil
}

func (s *Store) createTaxonomy(ctx context.Context, table, kind string, req TaxonomyIn) (Taxonomy, error) {
	if req.Name == nil || strings.TrimSpace(*req.Name) == "" {
		return Taxonomy{}, bad("name is required")
	}
	desc := *req.Name
	if req.DBDescription != nil {
		desc = *req.DBDescription
	}
	lang := "en"
	if req.LanguageCode != nil {
		lang = normalizeLang(*req.LanguageCode)
	}
	slug := slugify(*req.Name)
	if req.Slug != nil && strings.TrimSpace(*req.Slug) != "" {
		slug = strings.TrimSpace(*req.Slug)
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO `+table+` (db_description, usage_count) VALUES (?, 0)`, desc)
	if err != nil {
		return Taxonomy{}, err
	}
	id64, _ := res.LastInsertId()
	id := int(id64)
	if _, err := s.db.ExecContext(ctx, `INSERT INTO `+kind+`_i18n (`+kind+`_id, language_code, name, slug) VALUES (?, ?, ?, ?)`,
		id, lang, *req.Name, slug); err != nil {
		return Taxonomy{}, err
	}
	s.popCache.clear()
	items, err := s.taxonomyByIDs(ctx, table, kind, []int{id}, lang)
	if err != nil || len(items) == 0 {
		return Taxonomy{}, err
	}
	return items[0], nil
}

func (s *Store) updateTaxonomy(ctx context.Context, table, kind string, id int, req TaxonomyIn) (Taxonomy, error) {
	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM `+table+` WHERE id = ?`, id).Scan(&exists)
	if isNotFound(err) {
		label := "Category"
		if kind == "tag" {
			label = "Tag"
		}
		return Taxonomy{}, missing(label + " not found: " + itoa(id))
	}
	if err != nil {
		return Taxonomy{}, err
	}
	if req.DBDescription != nil {
		if _, err := s.db.ExecContext(ctx, `UPDATE `+table+` SET db_description = ? WHERE id = ?`, *req.DBDescription, id); err != nil {
			return Taxonomy{}, err
		}
	}
	lang := "en"
	if req.LanguageCode != nil {
		lang = normalizeLang(*req.LanguageCode)
	}
	var i18nID int
	err = s.db.QueryRowContext(ctx, `SELECT id FROM `+kind+`_i18n WHERE `+kind+`_id = ? AND language_code = ?`, id, lang).Scan(&i18nID)
	if isNotFound(err) {
		if req.Name == nil || strings.TrimSpace(*req.Name) == "" {
			return Taxonomy{}, bad("name is required when creating a translation")
		}
		slug := slugify(*req.Name)
		if req.Slug != nil && strings.TrimSpace(*req.Slug) != "" {
			slug = strings.TrimSpace(*req.Slug)
		}
		if _, err := s.db.ExecContext(ctx, `INSERT INTO `+kind+`_i18n (`+kind+`_id, language_code, name, slug) VALUES (?, ?, ?, ?)`,
			id, lang, strings.TrimSpace(*req.Name), slug); err != nil {
			return Taxonomy{}, err
		}
	} else if err != nil {
		return Taxonomy{}, err
	} else {
		if req.Name != nil {
			if _, err := s.db.ExecContext(ctx, `UPDATE `+kind+`_i18n SET name = ? WHERE id = ?`, *req.Name, i18nID); err != nil {
				return Taxonomy{}, err
			}
		}
		if req.Slug != nil {
			if _, err := s.db.ExecContext(ctx, `UPDATE `+kind+`_i18n SET slug = ? WHERE id = ?`, *req.Slug, i18nID); err != nil {
				return Taxonomy{}, err
			}
		} else if req.Name != nil {
			if _, err := s.db.ExecContext(ctx, `UPDATE `+kind+`_i18n SET slug = ? WHERE id = ?`, slugify(*req.Name), i18nID); err != nil {
				return Taxonomy{}, err
			}
		}
	}
	s.popCache.clear()
	items, err := s.taxonomyByIDs(ctx, table, kind, []int{id}, lang)
	if err != nil || len(items) == 0 {
		return Taxonomy{}, err
	}
	return items[0], nil
}

func (s *Store) deleteTaxonomy(ctx context.Context, table, kind, linkTable, linkCol string, id int, label string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM `+table+` WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return missing(label + " not found: " + itoa(id))
	}
	_, _ = s.db.ExecContext(ctx, `DELETE FROM `+kind+`_i18n WHERE `+kind+`_id = ?`, id)
	_, _ = s.db.ExecContext(ctx, `DELETE FROM `+linkTable+` WHERE `+linkCol+` = ?`, id)
	s.popCache.clear()
	return nil
}

func (s *Store) refreshCategoryCounts(ctx context.Context, ids []int) error {
	return s.refreshCounts(ctx, "categories", "posts_categories", "category_id", ids)
}

func (s *Store) refreshTagCounts(ctx context.Context, ids []int) error {
	return s.refreshCounts(ctx, "tags", "posts_tags", "tag_id", ids)
}

func (s *Store) refreshCounts(ctx context.Context, table, link, col string, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	for _, id := range ids {
		if _, err := s.db.ExecContext(ctx, `
			UPDATE `+table+` SET usage_count = (
				SELECT COUNT(*) FROM `+link+` WHERE `+col+` = ?
			) WHERE id = ?`, id, id); err != nil {
			return err
		}
	}
	s.popCache.clear()
	return nil
}

func (s *Store) linkIDs(ctx context.Context, postID string) ([]int, []int, error) {
	cats, err := queryIDs(ctx, s.db, `SELECT category_id FROM posts_categories WHERE post_id = ?`, postID)
	if err != nil {
		return nil, nil, err
	}
	tags, err := queryIDs(ctx, s.db, `SELECT tag_id FROM posts_tags WHERE post_id = ?`, postID)
	return cats, tags, err
}

func queryIDs(ctx context.Context, db *sql.DB, q, id string) ([]int, error) {
	rows, err := db.QueryContext(ctx, q, id)
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

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func ParseIDs(csv string) []int {
	if strings.TrimSpace(csv) == "" {
		return []int{}
	}
	out := []int{}
	for _, part := range strings.Split(csv, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n := 0
		ok := true
		for _, c := range part {
			if c < '0' || c > '9' {
				ok = false
				break
			}
			n = n*10 + int(c-'0')
		}
		if ok {
			out = append(out, n)
		}
	}
	return out
}
