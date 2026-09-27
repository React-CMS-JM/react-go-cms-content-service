package repo

import (
	"context"
	"database/sql"

	"react-go-cms-content-service/internal/application/service/settings"
	"react-go-cms-content-service/internal/domain/entity"
	"react-go-cms-content-service/internal/infrastructure/identity"
	"react-go-cms-content-service/internal/infrastructure/repository/model"
	"strings"
)

// SettingsRepository persists site settings in MySQL.
type SettingsRepository struct {
	db *sql.DB
}

// NewSettingsRepository builds a settings repository.
func NewSettingsRepository(db *sql.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

var _ settings.Repository = (*SettingsRepository)(nil)

// Setting returns a site setting or the fallback when the row is missing.
func (r *SettingsRepository) Setting(ctx context.Context, key, fallback string) string {
	var value sql.NullString
	var err error
	err = r.db.QueryRowContext(ctx, `SELECT setting_value FROM site_settings WHERE setting_key = ?`, key).Scan(&value)
	if err != nil || !value.Valid {
		return fallback
	}
	return value.String
}

// SettingI18n returns a localized setting, falling back to any language and then the default.
func (r *SettingsRepository) SettingI18n(ctx context.Context, key, lang, fallback string) string {
	var value string
	var err error
	err = r.db.QueryRowContext(ctx, `
		SELECT setting_value FROM site_setting_i18n WHERE setting_key = ? AND language_code = ? LIMIT 1`, key, lang).Scan(&value)
	if isNotFound(err) {
		err = r.db.QueryRowContext(ctx, `SELECT setting_value FROM site_setting_i18n WHERE setting_key = ? ORDER BY id LIMIT 1`, key).Scan(&value)
	}
	if err != nil {
		return fallback
	}
	return value
}

// UpsertSetting inserts or updates one site setting.
func (r *SettingsRepository) UpsertSetting(ctx context.Context, key, value string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO site_settings (setting_key, setting_value) VALUES (?, ?)
		ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value)`, key, value)
	return err
}

// UpsertSettingI18n ensures the setting key exists and upserts the translation.
func (r *SettingsRepository) UpsertSettingI18n(ctx context.Context, key, lang, value string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO site_settings (setting_key, setting_value) VALUES (?, NULL)
		ON DUPLICATE KEY UPDATE setting_key = setting_key`, key)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO site_setting_i18n (setting_key, language_code, setting_value) VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value)`, key, lang, value)
	return err
}

// ListHeroCtas returns hero calls to action with their labels.
func (r *SettingsRepository) ListHeroCtas(ctx context.Context) ([]entity.HeroCta, error) {
	var rows *sql.Rows
	var err error
	rows, err = r.db.QueryContext(ctx, `
		SELECT id, href, text_color, background_color, is_visible FROM hero_ctas ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []entity.HeroCta{}
	for rows.Next() {
		var row model.HeroCtaRow
		err = rows.Scan(&row.ID, &row.Href, &row.TextColor, &row.BackgroundColor, &row.IsVisible)
		if err != nil {
			return nil, err
		}
		cta := entity.HeroCta{
			ID:              row.ID,
			Visible:         row.IsVisible == 1,
			Href:            row.Href,
			TextColor:       row.TextColor,
			BackgroundColor: row.BackgroundColor,
			Labels:          map[string]string{},
		}
		var labelRows *sql.Rows
		labelRows, err = r.db.QueryContext(ctx, `SELECT language_code, label FROM hero_cta_i18n WHERE hero_cta_id = ?`, cta.ID)
		if err != nil {
			return nil, err
		}
		for labelRows.Next() {
			var lang, label string
			err = labelRows.Scan(&lang, &label)
			if err != nil {
				labelRows.Close()
				return nil, err
			}
			cta.Labels[lang] = label
		}
		err = labelRows.Err()
		labelRows.Close()
		if err != nil {
			return nil, err
		}
		out = append(out, cta)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ReplaceHeroCtas deletes the current hero calls to action and inserts the given rows.
func (r *SettingsRepository) ReplaceHeroCtas(ctx context.Context, ctas []entity.HeroCtaWrite) error {
	var tx *sql.Tx
	var err error
	tx, err = r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `DELETE FROM hero_cta_i18n`)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM hero_ctas`)
	if err != nil {
		return err
	}
	for index, cta := range ctas {
		id := cta.ID
		if id == "" {
			id = identity.NewUUID()
		}
		visible := 0
		if cta.Visible {
			visible = 1
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO hero_ctas (id, href, text_color, background_color, sort_order, is_visible)
			VALUES (?, ?, ?, ?, ?, ?)`, id, cta.Href, cta.TextColor, cta.BackgroundColor, index, visible)
		if err != nil {
			return err
		}
		for key, value := range cta.Labels {
			if strings.TrimSpace(key) == "" {
				continue
			}
			_, err = tx.ExecContext(ctx, `
				INSERT INTO hero_cta_i18n (hero_cta_id, language_code, label) VALUES (?, ?, ?)`,
				id, strings.ToLower(key), value)
			if err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// ListHomeSections returns raw home section rows.
func (r *SettingsRepository) ListHomeSections(ctx context.Context) ([]entity.HomeSectionRecord, error) {
	var rows *sql.Rows
	var err error
	rows, err = r.db.QueryContext(ctx, `SELECT section_key, is_visible, item_limit FROM home_sections ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []entity.HomeSectionRecord{}
	for rows.Next() {
		var row model.HomeSectionRow
		err = rows.Scan(&row.SectionKey, &row.IsVisible, &row.ItemLimit)
		if err != nil {
			return nil, err
		}
		out = append(out, entity.HomeSectionRecord{Key: row.SectionKey, Visible: row.IsVisible == 1, ItemLimit: row.ItemLimit})
	}
	return out, rows.Err()
}

// SaveHomeSection inserts or updates one home section.
func (r *SettingsRepository) SaveHomeSection(ctx context.Context, index int, id string, visible bool, limit int) error {
	var existing int
	var err error
	err = r.db.QueryRowContext(ctx, `SELECT id FROM home_sections WHERE section_key = ?`, id).Scan(&existing)
	if isNotFound(err) {
		_, err = r.db.ExecContext(ctx, `
				INSERT INTO home_sections (section_key, sort_order, is_visible, item_limit) VALUES (?, ?, ?, ?)`,
			id, index, boolInt(visible), limit)
		return err
	}
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
			UPDATE home_sections SET sort_order = ?, is_visible = ?, item_limit = ? WHERE id = ?`,
		index, boolInt(visible), limit, existing)
	return err
}

// ListMainMenu returns menu items. Item limits stay unset.
func (r *SettingsRepository) ListMainMenu(ctx context.Context) ([]entity.VisItem, error) {
	var rows *sql.Rows
	var err error
	rows, err = r.db.QueryContext(ctx, `SELECT item_key, is_visible FROM nav_items ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []entity.VisItem{}
	for rows.Next() {
		var row model.NavItemRow
		err = rows.Scan(&row.ItemKey, &row.IsVisible)
		if err != nil {
			return nil, err
		}
		out = append(out, entity.VisItem{ID: row.ItemKey, Visible: row.IsVisible == 1, ItemLimit: nil})
	}
	return out, rows.Err()
}

// SaveMenuItem inserts or updates one menu item.
func (r *SettingsRepository) SaveMenuItem(ctx context.Context, index int, id string, visible bool) error {
	var existing int
	var err error
	err = r.db.QueryRowContext(ctx, `SELECT id FROM nav_items WHERE item_key = ?`, id).Scan(&existing)
	if isNotFound(err) {
		_, err = r.db.ExecContext(ctx, `
				INSERT INTO nav_items (item_key, sort_order, is_visible) VALUES (?, ?, ?)`,
			id, index, boolInt(visible))
		return err
	}
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `UPDATE nav_items SET sort_order = ?, is_visible = ? WHERE id = ?`, index, boolInt(visible), existing)
	return err
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
