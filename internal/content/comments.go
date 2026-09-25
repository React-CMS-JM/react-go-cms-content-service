package content

import (
	"context"
	"database/sql"
	"strings"

	"react-go-cms-content-service/internal/platform"
)

type Comment struct {
	ID                            string             `json:"id"`
	PostID                        string             `json:"postId"`
	UserID                        string             `json:"userId"`
	ParentCommentID               *string            `json:"parentCommentId"`
	Content                       string             `json:"content"`
	LanguageCode                  string             `json:"languageCode"`
	AvailableTranslationLanguages []string           `json:"availableTranslationLanguages"`
	Status                        string             `json:"status"`
	CreatedAt                     platform.LocalJSON `json:"createdAt"`
	UpdatedAt                     platform.LocalJSON `json:"updatedAt"`
}

type CommentTranslation struct {
	CommentID    string `json:"commentId"`
	LanguageCode string `json:"languageCode"`
	Content      string `json:"content"`
	IsOriginal   bool   `json:"isOriginal"`
}

func (s *Store) ListComments(ctx context.Context, postID, status string) ([]Comment, error) {
	q := `SELECT id, post_id, user_id, parent_comment_id, status, created_at, updated_at FROM comments WHERE 1=1`
	args := []any{}
	if strings.TrimSpace(postID) != "" {
		q += ` AND post_id = ?`
		args = append(args, postID)
	}
	if strings.TrimSpace(status) != "" {
		q += ` AND status = ?`
		args = append(args, strings.TrimSpace(status))
	}
	q += ` ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type row struct {
		id, postID, userID, status string
		parent                     sql.NullString
		created, updated           sql.NullTime
	}
	list := []row{}
	ids := []any{}
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.postID, &r.userID, &r.parent, &r.status, &r.created, &r.updated); err != nil {
			return nil, err
		}
		list = append(list, r)
		ids = append(ids, r.id)
	}
	i18n, err := s.activeCommentI18n(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]Comment, 0, len(list))
	for _, r := range list {
		out = append(out, toComment(r.id, r.postID, r.userID, r.parent, r.status, r.created, r.updated, i18n[r.id]))
	}
	return out, nil
}

type cI18n struct {
	lang, content string
	original      bool
}

func (s *Store) activeCommentI18n(ctx context.Context, ids []any) (map[string][]cI18n, error) {
	out := map[string][]cI18n{}
	if len(ids) == 0 {
		return out, nil
	}
	q := `SELECT comment_id, language_code, content, is_original FROM comment_i18n
		WHERE marked_for_deletion = 0 AND comment_id IN (` + placeholders(len(ids)) + `)`
	rows, err := s.db.QueryContext(ctx, q, ids...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var row cI18n
		var orig int
		if err := rows.Scan(&id, &row.lang, &row.content, &orig); err != nil {
			return nil, err
		}
		row.original = orig == 1
		out[id] = append(out[id], row)
	}
	return out, rows.Err()
}

func toComment(id, postID, userID string, parent sql.NullString, status string, created, updated sql.NullTime, rows []cI18n) Comment {
	c := Comment{
		ID:                            id,
		PostID:                        postID,
		UserID:                        userID,
		ParentCommentID:               platform.StrPtr(parent),
		Status:                        status,
		CreatedAt:                     platform.LocalFrom(created),
		UpdatedAt:                     platform.LocalFrom(updated),
		AvailableTranslationLanguages: []string{},
		Content:                       "",
		LanguageCode:                  "en",
	}
	var original *cI18n
	for i := range rows {
		if rows[i].original && original == nil {
			original = &rows[i]
		} else if !rows[i].original {
			c.AvailableTranslationLanguages = append(c.AvailableTranslationLanguages, rows[i].lang)
		}
	}
	if original == nil && len(rows) > 0 {
		original = &rows[0]
	}
	if original != nil {
		c.Content = original.content
		c.LanguageCode = original.lang
	}
	return c
}

type CreateComment struct {
	PostID          string  `json:"postId"`
	UserID          *string `json:"userId"`
	ParentCommentID *string `json:"parentCommentId"`
	Content         string  `json:"content"`
	LanguageCode    *string `json:"languageCode"`
	Status          *string `json:"status"`
}

func (s *Store) CreateComment(ctx context.Context, req CreateComment, fallbackUser string) (Comment, error) {
	if strings.TrimSpace(req.PostID) == "" {
		return Comment{}, bad("postId is required")
	}
	if strings.TrimSpace(req.Content) == "" {
		return Comment{}, bad("content is required")
	}
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM posts WHERE id = ?`, req.PostID).Scan(&exists); isNotFound(err) {
		return Comment{}, missing("Post not found: " + req.PostID)
	} else if err != nil {
		return Comment{}, err
	}
	userID := fallbackUser
	if req.UserID != nil && strings.TrimSpace(*req.UserID) != "" {
		userID = strings.TrimSpace(*req.UserID)
	}
	if strings.TrimSpace(userID) == "" {
		return Comment{}, bad("userId is required")
	}
	lang := "en"
	if req.LanguageCode != nil {
		lang = normalizeLang(*req.LanguageCode)
	}
	status := "approved"
	if req.Status != nil && strings.TrimSpace(*req.Status) != "" {
		status = *req.Status
	}
	id := platform.NewUUID()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Comment{}, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comments (id, post_id, user_id, parent_comment_id, status) VALUES (?, ?, ?, ?, ?)`,
		id, req.PostID, userID, req.ParentCommentID, status); err != nil {
		return Comment{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comment_i18n (comment_id, language_code, content, is_original, marked_for_deletion)
		VALUES (?, ?, ?, 1, 0)`, id, lang, req.Content); err != nil {
		return Comment{}, err
	}
	if err := tx.Commit(); err != nil {
		return Comment{}, err
	}
	return s.commentByID(ctx, id)
}

func (s *Store) UpdateComment(ctx context.Context, id, content string, lang *string) (Comment, error) {
	if strings.TrimSpace(content) == "" {
		return Comment{}, bad("content is required")
	}
	if err := s.requireComment(ctx, id); err != nil {
		return Comment{}, err
	}
	language := "en"
	if lang != nil {
		language = normalizeLang(*lang)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Comment{}, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE comment_i18n SET marked_for_deletion = 1 WHERE comment_id = ? AND marked_for_deletion = 0`, id); err != nil {
		return Comment{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comment_i18n (comment_id, language_code, content, is_original, marked_for_deletion)
		VALUES (?, ?, ?, 1, 0)`, id, language, content); err != nil {
		return Comment{}, err
	}
	if err := tx.Commit(); err != nil {
		return Comment{}, err
	}
	return s.commentByID(ctx, id)
}

func (s *Store) GetCommentTranslation(ctx context.Context, id, lang string) (CommentTranslation, error) {
	if err := s.requireComment(ctx, id); err != nil {
		return CommentTranslation{}, err
	}
	language := normalizeLang(lang)
	var content string
	var original int
	var gotLang string
	err := s.db.QueryRowContext(ctx, `
		SELECT language_code, content, is_original FROM comment_i18n
		WHERE comment_id = ? AND language_code = ? AND marked_for_deletion = 0 LIMIT 1`, id, language).
		Scan(&gotLang, &content, &original)
	if isNotFound(err) {
		return CommentTranslation{}, missing("Translation not found for comment " + id + " lang=" + language)
	}
	if err != nil {
		return CommentTranslation{}, err
	}
	return CommentTranslation{CommentID: id, LanguageCode: gotLang, Content: content, IsOriginal: original == 1}, nil
}

func (s *Store) UpsertCommentTranslation(ctx context.Context, id, lang, content string) (CommentTranslation, error) {
	if strings.TrimSpace(content) == "" {
		return CommentTranslation{}, bad("content is required")
	}
	if err := s.requireComment(ctx, id); err != nil {
		return CommentTranslation{}, err
	}
	language := normalizeLang(lang)
	var rowID int
	var original int
	err := s.db.QueryRowContext(ctx, `
		SELECT id, is_original FROM comment_i18n
		WHERE comment_id = ? AND language_code = ? AND marked_for_deletion = 0 LIMIT 1`, id, language).Scan(&rowID, &original)
	if isNotFound(err) {
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO comment_i18n (comment_id, language_code, content, is_original, marked_for_deletion)
			VALUES (?, ?, ?, 0, 0)`, id, language, content); err != nil {
			return CommentTranslation{}, err
		}
		return CommentTranslation{CommentID: id, LanguageCode: language, Content: content, IsOriginal: false}, nil
	}
	if err != nil {
		return CommentTranslation{}, err
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE comment_i18n SET content = ? WHERE id = ?`, content, rowID); err != nil {
		return CommentTranslation{}, err
	}
	return CommentTranslation{CommentID: id, LanguageCode: language, Content: content, IsOriginal: original == 1}, nil
}

func (s *Store) PatchCommentStatus(ctx context.Context, id, status string) (Comment, error) {
	if strings.TrimSpace(status) == "" {
		return Comment{}, bad("status is required")
	}
	res, err := s.db.ExecContext(ctx, `UPDATE comments SET status = ? WHERE id = ?`, strings.TrimSpace(status), id)
	if err != nil {
		return Comment{}, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return Comment{}, missing("Comment not found: " + id)
	}
	return s.commentByID(ctx, id)
}

func (s *Store) DeleteComment(ctx context.Context, id string) error {
	if err := s.requireComment(ctx, id); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM comment_i18n WHERE comment_id = ?`, id); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM comments WHERE id = ?`, id)
	return err
}

func (s *Store) requireComment(ctx context.Context, id string) error {
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM comments WHERE id = ?`, id).Scan(&one)
	if isNotFound(err) {
		return missing("Comment not found: " + id)
	}
	return err
}

func (s *Store) commentByID(ctx context.Context, id string) (Comment, error) {
	var postID, userID, status string
	var parent sql.NullString
	var created, updated sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT post_id, user_id, parent_comment_id, status, created_at, updated_at FROM comments WHERE id = ?`, id).
		Scan(&postID, &userID, &parent, &status, &created, &updated)
	if isNotFound(err) {
		return Comment{}, missing("Comment not found: " + id)
	}
	if err != nil {
		return Comment{}, err
	}
	i18n, err := s.activeCommentI18n(ctx, []any{id})
	if err != nil {
		return Comment{}, err
	}
	return toComment(id, postID, userID, parent, status, created, updated, i18n[id]), nil
}
