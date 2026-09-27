package repo

import (
	"context"
	"database/sql"
	"strings"

	"react-go-cms-content-service/internal/application/service/uistring"
	"react-go-cms-content-service/internal/domain/apperror"
	"react-go-cms-content-service/internal/domain/entity"
	"react-go-cms-content-service/internal/infrastructure/repository/model"
)

// UIStringRepository persists UI strings in MySQL.
type UIStringRepository struct {
	db *sql.DB
}

// NewUIStringRepository builds a UI-string repository.
func NewUIStringRepository(db *sql.DB) *UIStringRepository {
	return &UIStringRepository{db: db}
}

var _ uistring.Repository = (*UIStringRepository)(nil)

// List returns UI strings that have a translation in the requested language.
func (r *UIStringRepository) List(ctx context.Context, language, component string) ([]entity.UIString, error) {
	query := `SELECT string_key, ui_component FROM param_ui_strings`
	args := []any{}
	if strings.TrimSpace(component) != "" {
		query += ` WHERE ui_component = ?`
		args = append(args, strings.TrimSpace(component))
	} else {
		query += ` WHERE ui_component <> 'AdminSidebar'`
	}
	var rows *sql.Rows
	var err error
	rows, err = r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := []model.UIStringKeyRow{}
	for rows.Next() {
		var key model.UIStringKeyRow
		err = rows.Scan(&key.StringKey, &key.UIComponent)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	out := []entity.UIString{}
	for _, key := range keys {
		var value string
		var updated sql.NullTime
		var storedLang string
		err = r.db.QueryRowContext(ctx, `
			SELECT language_code, string_value, updated_at FROM param_ui_string_i18n
			WHERE string_key = ? AND language_code = ?`, key.StringKey, language).Scan(&storedLang, &value, &updated)
		if isNotFound(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, entity.UIString{
			StringKey:    key.StringKey,
			UIComponent:  StrPtr(key.UIComponent),
			LanguageCode: storedLang,
			StringValue:  value,
			UpdatedAt:    LocalFrom(updated),
		})
	}
	return out, nil
}

// Upsert writes one translation for an existing UI string key.
func (r *UIStringRepository) Upsert(ctx context.Context, stringKey, language, value string) (entity.UIString, error) {
	var component sql.NullString
	var err error
	err = r.db.QueryRowContext(ctx, `SELECT ui_component FROM param_ui_strings WHERE string_key = ?`, stringKey).Scan(&component)
	if isNotFound(err) {
		return entity.UIString{}, apperror.NotFound("UI string key not found: " + stringKey)
	}
	if err != nil {
		return entity.UIString{}, err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO param_ui_string_i18n (string_key, language_code, string_value) VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE string_value = VALUES(string_value)`, stringKey, language, value)
	if err != nil {
		return entity.UIString{}, err
	}
	var updated sql.NullTime
	var stored string
	err = r.db.QueryRowContext(ctx, `
		SELECT string_value, updated_at FROM param_ui_string_i18n WHERE string_key = ? AND language_code = ?`,
		stringKey, language).Scan(&stored, &updated)
	if err != nil {
		return entity.UIString{}, err
	}
	return entity.UIString{
		StringKey:    stringKey,
		UIComponent:  StrPtr(component),
		LanguageCode: language,
		StringValue:  stored,
		UpdatedAt:    LocalFrom(updated),
	}, nil
}
