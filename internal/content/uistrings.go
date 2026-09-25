package content

import (
	"context"
	"database/sql"
	"strings"

	"react-go-cms-content-service/internal/platform"
)

type UIString struct {
	StringKey    string             `json:"stringKey"`
	UIComponent  *string            `json:"uiComponent"`
	LanguageCode string             `json:"languageCode"`
	StringValue  string             `json:"stringValue"`
	UpdatedAt    platform.LocalJSON `json:"updatedAt"`
}

func (s *Store) ListUIStrings(ctx context.Context, lang, component string) ([]UIString, error) {
	language := normalizeLang(lang)
	q := `SELECT string_key, ui_component FROM param_ui_strings`
	args := []any{}
	if strings.TrimSpace(component) != "" {
		q += ` WHERE ui_component = ?`
		args = append(args, strings.TrimSpace(component))
	} else {
		q += ` WHERE ui_component <> 'AdminSidebar'`
	}
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type keyRow struct {
		key string
		ui  sql.NullString
	}
	keys := []keyRow{}
	for rows.Next() {
		var k keyRow
		if err := rows.Scan(&k.key, &k.ui); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	out := []UIString{}
	for _, k := range keys {
		var val string
		var updated sql.NullTime
		var gotLang string
		err := s.db.QueryRowContext(ctx, `
			SELECT language_code, string_value, updated_at FROM param_ui_string_i18n
			WHERE string_key = ? AND language_code = ?`, k.key, language).Scan(&gotLang, &val, &updated)
		if isNotFound(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, UIString{
			StringKey:    k.key,
			UIComponent:  platform.StrPtr(k.ui),
			LanguageCode: gotLang,
			StringValue:  val,
			UpdatedAt:    platform.LocalFrom(updated),
		})
	}
	return out, nil
}

func (s *Store) UpsertUIString(ctx context.Context, stringKey, languageCode, value string) (UIString, error) {
	if strings.TrimSpace(languageCode) == "" {
		return UIString{}, bad("languageCode is required")
	}
	var ui sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT ui_component FROM param_ui_strings WHERE string_key = ?`, stringKey).Scan(&ui)
	if isNotFound(err) {
		return UIString{}, missing("UI string key not found: " + stringKey)
	}
	if err != nil {
		return UIString{}, err
	}
	lang := normalizeLang(languageCode)
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO param_ui_string_i18n (string_key, language_code, string_value) VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE string_value = VALUES(string_value)`, stringKey, lang, value); err != nil {
		return UIString{}, err
	}
	var updated sql.NullTime
	var stored string
	if err := s.db.QueryRowContext(ctx, `
		SELECT string_value, updated_at FROM param_ui_string_i18n WHERE string_key = ? AND language_code = ?`,
		stringKey, lang).Scan(&stored, &updated); err != nil {
		return UIString{}, err
	}
	return UIString{
		StringKey:    stringKey,
		UIComponent:  platform.StrPtr(ui),
		LanguageCode: lang,
		StringValue:  stored,
		UpdatedAt:    platform.LocalFrom(updated),
	}, nil
}
