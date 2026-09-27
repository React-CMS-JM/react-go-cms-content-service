package repo

import (
	"context"
	"database/sql"
	"strings"

	"react-go-cms-content-service/internal/application/service/comment"
	"react-go-cms-content-service/internal/domain/apperror"
	"react-go-cms-content-service/internal/domain/entity"
	"react-go-cms-content-service/internal/infrastructure/identity"
	"react-go-cms-content-service/internal/infrastructure/repository/model"
)

// CommentRepository persists comments in MySQL.
type CommentRepository struct {
	db *sql.DB
}

// NewCommentRepository builds a comment repository.
func NewCommentRepository(db *sql.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

var _ comment.Repository = (*CommentRepository)(nil)

// List returns comments matching the optional post and status filters.
func (r *CommentRepository) List(ctx context.Context, postID, status string) ([]entity.CommentRecord, error) {
	query := `SELECT id, post_id, user_id, parent_comment_id, status, created_at, updated_at FROM comments WHERE 1=1`
	args := []any{}
	if strings.TrimSpace(postID) != "" {
		query += ` AND post_id = ?`
		args = append(args, postID)
	}
	if strings.TrimSpace(status) != "" {
		query += ` AND status = ?`
		args = append(args, strings.TrimSpace(status))
	}
	query += ` ORDER BY created_at DESC`
	var rows *sql.Rows
	var err error
	rows, err = r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []model.CommentRow{}
	ids := []any{}
	for rows.Next() {
		var row model.CommentRow
		err = rows.Scan(&row.ID, &row.PostID, &row.UserID, &row.ParentCommentID, &row.Status, &row.CreatedAt, &row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, row)
		ids = append(ids, row.ID)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	var translations map[string][]entity.CommentText
	translations, err = r.activeTranslations(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]entity.CommentRecord, 0, len(list))
	for _, row := range list {
		out = append(out, toCommentRecord(row, translations[row.ID]))
	}
	return out, nil
}

// PostExists reports whether a post row is present.
func (r *CommentRepository) PostExists(ctx context.Context, postID string) (bool, error) {
	var exists int
	var err error
	err = r.db.QueryRowContext(ctx, `SELECT 1 FROM posts WHERE id = ?`, postID).Scan(&exists)
	if isNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Insert stores a comment and its original translation.
func (r *CommentRepository) Insert(ctx context.Context, postID, userID string, parent *string, status, lang, content string) (string, error) {
	var id string
	id = identity.NewUUID()
	var tx *sql.Tx
	var err error
	tx, err = r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO comments (id, post_id, user_id, parent_comment_id, status) VALUES (?, ?, ?, ?, ?)`,
		id, postID, userID, parent, status)
	if err != nil {
		return "", err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO comment_i18n (comment_id, language_code, content, is_original, marked_for_deletion)
		VALUES (?, ?, ?, 1, 0)`, id, lang, content)
	if err != nil {
		return "", err
	}
	err = tx.Commit()
	if err != nil {
		return "", err
	}
	return id, nil
}

// Require returns not found when the comment is missing.
func (r *CommentRepository) Require(ctx context.Context, id string) error {
	var one int
	var err error
	err = r.db.QueryRowContext(ctx, `SELECT 1 FROM comments WHERE id = ?`, id).Scan(&one)
	if isNotFound(err) {
		return apperror.NotFound("Comment not found: " + id)
	}
	return err
}

// ReplaceContent marks current translations deleted and inserts a new original.
func (r *CommentRepository) ReplaceContent(ctx context.Context, id, lang, content string) error {
	var tx *sql.Tx
	var err error
	tx, err = r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `UPDATE comment_i18n SET marked_for_deletion = 1 WHERE comment_id = ? AND marked_for_deletion = 0`, id)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO comment_i18n (comment_id, language_code, content, is_original, marked_for_deletion)
		VALUES (?, ?, ?, 1, 0)`, id, lang, content)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// ReadTranslation loads one active translation. found is false when the row is absent.
func (r *CommentRepository) ReadTranslation(ctx context.Context, id, lang string) (string, string, bool, bool, error) {
	var storedLang string
	var content string
	var original int
	var err error
	err = r.db.QueryRowContext(ctx, `
		SELECT language_code, content, is_original FROM comment_i18n
		WHERE comment_id = ? AND language_code = ? AND marked_for_deletion = 0 LIMIT 1`, id, lang).
		Scan(&storedLang, &content, &original)
	if isNotFound(err) {
		return "", "", false, false, nil
	}
	if err != nil {
		return "", "", false, false, err
	}
	return storedLang, content, original == 1, true, nil
}

// FindTranslation loads the row id and original flag for one active translation.
func (r *CommentRepository) FindTranslation(ctx context.Context, id, lang string) (int, bool, bool, error) {
	var rowID int
	var original int
	var err error
	err = r.db.QueryRowContext(ctx, `
		SELECT id, is_original FROM comment_i18n
		WHERE comment_id = ? AND language_code = ? AND marked_for_deletion = 0 LIMIT 1`, id, lang).Scan(&rowID, &original)
	if isNotFound(err) {
		return 0, false, false, nil
	}
	if err != nil {
		return 0, false, false, err
	}
	return rowID, original == 1, true, nil
}

// InsertTranslation adds a non-original translation.
func (r *CommentRepository) InsertTranslation(ctx context.Context, id, lang, content string) error {
	_, err := r.db.ExecContext(ctx, `
			INSERT INTO comment_i18n (comment_id, language_code, content, is_original, marked_for_deletion)
			VALUES (?, ?, ?, 0, 0)`, id, lang, content)
	return err
}

// UpdateTranslation sets the content of one translation row.
func (r *CommentRepository) UpdateTranslation(ctx context.Context, rowID int, content string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE comment_i18n SET content = ? WHERE id = ?`, content, rowID)
	return err
}

// PatchStatus updates the comment status. False means no row matched.
func (r *CommentRepository) PatchStatus(ctx context.Context, id, status string) (bool, error) {
	var result sql.Result
	var err error
	result, err = r.db.ExecContext(ctx, `UPDATE comments SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return false, err
	}
	affected, _ := result.RowsAffected()
	return affected != 0, nil
}

// Delete removes translations and then the comment.
func (r *CommentRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM comment_i18n WHERE comment_id = ?`, id)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `DELETE FROM comments WHERE id = ?`, id)
	return err
}

// Find loads one comment and its active translations.
func (r *CommentRepository) Find(ctx context.Context, id string) (entity.CommentRecord, error) {
	var row model.CommentRow
	var err error
	err = r.db.QueryRowContext(ctx, `
		SELECT post_id, user_id, parent_comment_id, status, created_at, updated_at FROM comments WHERE id = ?`, id).
		Scan(&row.PostID, &row.UserID, &row.ParentCommentID, &row.Status, &row.CreatedAt, &row.UpdatedAt)
	if isNotFound(err) {
		return entity.CommentRecord{}, apperror.NotFound("Comment not found: " + id)
	}
	if err != nil {
		return entity.CommentRecord{}, err
	}
	row.ID = id
	var translations map[string][]entity.CommentText
	translations, err = r.activeTranslations(ctx, []any{id})
	if err != nil {
		return entity.CommentRecord{}, err
	}
	return toCommentRecord(row, translations[id]), nil
}

func (r *CommentRepository) activeTranslations(ctx context.Context, ids []any) (map[string][]entity.CommentText, error) {
	out := map[string][]entity.CommentText{}
	if len(ids) == 0 {
		return out, nil
	}
	query := `SELECT comment_id, language_code, content, is_original FROM comment_i18n
		WHERE marked_for_deletion = 0 AND comment_id IN (` + placeholders(len(ids)) + `)`
	var rows *sql.Rows
	var err error
	rows, err = r.db.QueryContext(ctx, query, ids...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var row model.CommentI18nRow
		err = rows.Scan(&id, &row.LanguageCode, &row.Content, &row.IsOriginal)
		if err != nil {
			return nil, err
		}
		out[id] = append(out[id], entity.CommentText{
			LanguageCode: row.LanguageCode,
			Content:      row.Content,
			Original:     row.IsOriginal == 1,
		})
	}
	return out, rows.Err()
}

func toCommentRecord(row model.CommentRow, translations []entity.CommentText) entity.CommentRecord {
	return entity.CommentRecord{
		ID:              row.ID,
		PostID:          row.PostID,
		UserID:          row.UserID,
		ParentCommentID: StrPtr(row.ParentCommentID),
		Status:          row.Status,
		CreatedAt:       LocalFrom(row.CreatedAt),
		UpdatedAt:       LocalFrom(row.UpdatedAt),
		Translations:    translations,
	}
}
