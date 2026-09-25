package content

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"react-go-cms-content-service/internal/platform"
)

type SiteSettings struct {
	SiteName        string    `json:"siteName"`
	SiteIconURL     string    `json:"siteIconUrl"`
	SiteDescription string    `json:"siteDescription"`
	PostsPerPage    int       `json:"postsPerPage"`
	HomeHero        HomeHero  `json:"homeHero"`
	HomeSections    []VisItem `json:"homeSections"`
	MainMenu        []VisItem `json:"mainMenu"`
}

type HomeHero struct {
	TitleVisible    bool      `json:"titleVisible"`
	SubtitleVisible bool      `json:"subtitleVisible"`
	Ctas            []HeroCta `json:"ctas"`
}

type HeroCta struct {
	ID              string            `json:"id"`
	Visible         bool              `json:"visible"`
	Href            string            `json:"href"`
	TextColor       string            `json:"textColor"`
	BackgroundColor string            `json:"backgroundColor"`
	Labels          map[string]string `json:"labels"`
}

type VisItem struct {
	ID        string `json:"id"`
	Visible   bool   `json:"visible"`
	ItemLimit *int   `json:"itemLimit"`
}

type SiteSettingsIn struct {
	SiteName        string       `json:"siteName"`
	SiteIconURL     *string      `json:"siteIconUrl"`
	SiteDescription *string      `json:"siteDescription"`
	PostsPerPage    int          `json:"postsPerPage"`
	HomeHero        *HomeHeroIn  `json:"homeHero"`
	HomeSections    *[]VisItemIn `json:"homeSections"`
	MainMenu        *[]VisItemIn `json:"mainMenu"`
}

type HomeHeroIn struct {
	TitleVisible    *bool       `json:"titleVisible"`
	SubtitleVisible *bool       `json:"subtitleVisible"`
	Ctas            []HeroCtaIn `json:"ctas"`
}

type HeroCtaIn struct {
	ID              string            `json:"id"`
	Visible         *bool             `json:"visible"`
	Href            *string           `json:"href"`
	TextColor       *string           `json:"textColor"`
	BackgroundColor *string           `json:"backgroundColor"`
	Labels          map[string]string `json:"labels"`
}

type VisItemIn struct {
	ID        string `json:"id"`
	Visible   *bool  `json:"visible"`
	ItemLimit *int   `json:"itemLimit"`
}

func (s *Store) GetSettings(ctx context.Context, lang string) (SiteSettings, error) {
	language := normalizeLang(lang)
	hero, err := s.GetHomeHero(ctx)
	if err != nil {
		return SiteSettings{}, err
	}
	sections, err := s.GetHomeSections(ctx)
	if err != nil {
		return SiteSettings{}, err
	}
	menu, err := s.GetMainMenu(ctx)
	if err != nil {
		return SiteSettings{}, err
	}
	ppp, _ := strconv.Atoi(s.setting(ctx, "posts_per_page", "9"))
	if ppp == 0 {
		ppp = 9
	}
	return SiteSettings{
		SiteName:        s.setting(ctx, "site_name", "React CMS"),
		SiteIconURL:     s.setting(ctx, "site_icon_url", ""),
		SiteDescription: s.settingI18n(ctx, "site_seo", language, ""),
		PostsPerPage:    ppp,
		HomeHero:        hero,
		HomeSections:    sections,
		MainMenu:        menu,
	}, nil
}

func (s *Store) PutSettings(ctx context.Context, body SiteSettingsIn, lang string) (SiteSettings, error) {
	language := normalizeLang(lang)
	if err := s.upsertSetting(ctx, "site_name", body.SiteName); err != nil {
		return SiteSettings{}, err
	}
	icon := ""
	if body.SiteIconURL != nil {
		icon = *body.SiteIconURL
	}
	if err := s.upsertSetting(ctx, "site_icon_url", icon); err != nil {
		return SiteSettings{}, err
	}
	ppp := body.PostsPerPage
	if ppp <= 0 {
		ppp = 9
	}
	if err := s.upsertSetting(ctx, "posts_per_page", strconv.Itoa(ppp)); err != nil {
		return SiteSettings{}, err
	}
	if body.SiteDescription != nil {
		if err := s.upsertSettingI18n(ctx, "site_seo", language, *body.SiteDescription); err != nil {
			return SiteSettings{}, err
		}
	}
	if body.HomeHero != nil {
		if _, err := s.PutHomeHero(ctx, *body.HomeHero); err != nil {
			return SiteSettings{}, err
		}
	}
	if body.HomeSections != nil {
		if _, err := s.PutHomeSections(ctx, *body.HomeSections); err != nil {
			return SiteSettings{}, err
		}
	}
	if body.MainMenu != nil {
		if _, err := s.PutMainMenu(ctx, *body.MainMenu); err != nil {
			return SiteSettings{}, err
		}
	}
	return s.GetSettings(ctx, language)
}

func (s *Store) GetHomeHero(ctx context.Context) (HomeHero, error) {
	hero := HomeHero{
		TitleVisible:    strings.EqualFold(s.setting(ctx, "hero_title_visible", "true"), "true"),
		SubtitleVisible: strings.EqualFold(s.setting(ctx, "hero_subtitle_visible", "true"), "true"),
		Ctas:            []HeroCta{},
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, href, text_color, background_color, is_visible FROM hero_ctas ORDER BY sort_order, id`)
	if err != nil {
		return hero, err
	}
	defer rows.Close()
	for rows.Next() {
		var c HeroCta
		var vis int
		if err := rows.Scan(&c.ID, &c.Href, &c.TextColor, &c.BackgroundColor, &vis); err != nil {
			return hero, err
		}
		c.Visible = vis == 1
		c.Labels = map[string]string{}
		lrows, err := s.db.QueryContext(ctx, `SELECT language_code, label FROM hero_cta_i18n WHERE hero_cta_id = ?`, c.ID)
		if err != nil {
			return hero, err
		}
		for lrows.Next() {
			var lang, label string
			if err := lrows.Scan(&lang, &label); err != nil {
				lrows.Close()
				return hero, err
			}
			c.Labels[lang] = label
		}
		lrows.Close()
		hero.Ctas = append(hero.Ctas, c)
	}
	return hero, rows.Err()
}

func (s *Store) PutHomeHero(ctx context.Context, hero HomeHeroIn) (HomeHero, error) {
	title := true
	if hero.TitleVisible != nil {
		title = *hero.TitleVisible
	}
	sub := true
	if hero.SubtitleVisible != nil {
		sub = *hero.SubtitleVisible
	}
	if err := s.upsertSetting(ctx, "hero_title_visible", strconv.FormatBool(title)); err != nil {
		return HomeHero{}, err
	}
	if err := s.upsertSetting(ctx, "hero_subtitle_visible", strconv.FormatBool(sub)); err != nil {
		return HomeHero{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return HomeHero{}, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM hero_cta_i18n`); err != nil {
		return HomeHero{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM hero_ctas`); err != nil {
		return HomeHero{}, err
	}
	for i, cta := range hero.Ctas {
		id := strings.TrimSpace(cta.ID)
		if id == "" {
			id = platform.NewUUID()
		}
		href := "/"
		if cta.Href != nil {
			href = *cta.Href
		}
		text := "#ffffff"
		if cta.TextColor != nil {
			text = *cta.TextColor
		}
		bg := "#6366f1"
		if cta.BackgroundColor != nil {
			bg = *cta.BackgroundColor
		}
		vis := 1
		if cta.Visible != nil && !*cta.Visible {
			vis = 0
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO hero_ctas (id, href, text_color, background_color, sort_order, is_visible)
			VALUES (?, ?, ?, ?, ?, ?)`, id, href, text, bg, i, vis); err != nil {
			return HomeHero{}, err
		}
		for k, v := range cta.Labels {
			if strings.TrimSpace(k) == "" {
				continue
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO hero_cta_i18n (hero_cta_id, language_code, label) VALUES (?, ?, ?)`,
				id, strings.ToLower(k), v); err != nil {
				return HomeHero{}, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return HomeHero{}, err
	}
	return s.GetHomeHero(ctx)
}

func (s *Store) GetHomeSections(ctx context.Context) ([]VisItem, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT section_key, is_visible, item_limit FROM home_sections ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []VisItem{}
	for rows.Next() {
		var key string
		var vis, limit int
		if err := rows.Scan(&key, &vis, &limit); err != nil {
			return nil, err
		}
		n := normalizeLimit(limit, defaultLimit(key))
		out = append(out, VisItem{ID: key, Visible: vis == 1, ItemLimit: &n})
	}
	return out, rows.Err()
}

func (s *Store) PutHomeSections(ctx context.Context, items []VisItemIn) ([]VisItem, error) {
	for i, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			continue
		}
		vis := true
		if item.Visible != nil {
			vis = *item.Visible
		}
		limit := normalizeLimitPtr(item.ItemLimit, defaultLimit(item.ID))
		var id int
		err := s.db.QueryRowContext(ctx, `SELECT id FROM home_sections WHERE section_key = ?`, item.ID).Scan(&id)
		if isNotFound(err) {
			if _, err := s.db.ExecContext(ctx, `
				INSERT INTO home_sections (section_key, sort_order, is_visible, item_limit) VALUES (?, ?, ?, ?)`,
				item.ID, i, boolInt(vis), limit); err != nil {
				return nil, err
			}
			continue
		}
		if err != nil {
			return nil, err
		}
		if _, err := s.db.ExecContext(ctx, `
			UPDATE home_sections SET sort_order = ?, is_visible = ?, item_limit = ? WHERE id = ?`,
			i, boolInt(vis), limit, id); err != nil {
			return nil, err
		}
	}
	return s.GetHomeSections(ctx)
}

func (s *Store) GetMainMenu(ctx context.Context) ([]VisItem, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT item_key, is_visible FROM nav_items ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []VisItem{}
	for rows.Next() {
		var key string
		var vis int
		if err := rows.Scan(&key, &vis); err != nil {
			return nil, err
		}
		out = append(out, VisItem{ID: key, Visible: vis == 1, ItemLimit: nil})
	}
	return out, rows.Err()
}

func (s *Store) PutMainMenu(ctx context.Context, items []VisItemIn) ([]VisItem, error) {
	for i, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			continue
		}
		vis := true
		if item.Visible != nil {
			vis = *item.Visible
		}
		var id int
		err := s.db.QueryRowContext(ctx, `SELECT id FROM nav_items WHERE item_key = ?`, item.ID).Scan(&id)
		if isNotFound(err) {
			if _, err := s.db.ExecContext(ctx, `
				INSERT INTO nav_items (item_key, sort_order, is_visible) VALUES (?, ?, ?)`,
				item.ID, i, boolInt(vis)); err != nil {
				return nil, err
			}
			continue
		}
		if err != nil {
			return nil, err
		}
		if _, err := s.db.ExecContext(ctx, `UPDATE nav_items SET sort_order = ?, is_visible = ? WHERE id = ?`, i, boolInt(vis), id); err != nil {
			return nil, err
		}
	}
	return s.GetMainMenu(ctx)
}

func (s *Store) setting(ctx context.Context, key, def string) string {
	var val sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT setting_value FROM site_settings WHERE setting_key = ?`, key).Scan(&val)
	if err != nil || !val.Valid {
		return def
	}
	return val.String
}

func (s *Store) settingI18n(ctx context.Context, key, lang, def string) string {
	var val string
	err := s.db.QueryRowContext(ctx, `
		SELECT setting_value FROM site_setting_i18n WHERE setting_key = ? AND language_code = ? LIMIT 1`, key, lang).Scan(&val)
	if isNotFound(err) {
		err = s.db.QueryRowContext(ctx, `SELECT setting_value FROM site_setting_i18n WHERE setting_key = ? ORDER BY id LIMIT 1`, key).Scan(&val)
	}
	if err != nil {
		return def
	}
	return val
}

func (s *Store) upsertSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO site_settings (setting_key, setting_value) VALUES (?, ?)
		ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value)`, key, value)
	return err
}

func (s *Store) upsertSettingI18n(ctx context.Context, key, lang, value string) error {
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO site_settings (setting_key, setting_value) VALUES (?, NULL)
		ON DUPLICATE KEY UPDATE setting_key = setting_key`, key); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO site_setting_i18n (setting_key, language_code, setting_value) VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value)`, key, lang, value)
	return err
}

func defaultLimit(section string) int {
	switch strings.ToLower(strings.TrimSpace(section)) {
	case "services":
		return 5
	case "products":
		return 6
	case "blog":
		return 9
	case "courses":
		return 3
	default:
		return 6
	}
}

func normalizeLimit(v, fallback int) int {
	if v < 1 {
		return 1
	}
	if v > 50 {
		return 50
	}
	return v
}

func normalizeLimitPtr(v *int, fallback int) int {
	if v == nil {
		return normalizeLimit(fallback, fallback)
	}
	return normalizeLimit(*v, fallback)
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
