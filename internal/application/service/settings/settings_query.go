package settings

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"react-go-cms-content-service/internal/domain/entity"
)

// Get returns the site settings document for one language.
func (s *Service) Get(ctx context.Context, lang string) (entity.SiteSettings, error) {
	language := normalizeLang(lang)
	var hero entity.HomeHero
	var err error
	hero, err = s.Hero(ctx)
	if err != nil {
		return entity.SiteSettings{}, err
	}
	var sections []entity.VisItem
	sections, err = s.HomeSections(ctx)
	if err != nil {
		return entity.SiteSettings{}, err
	}
	var menu []entity.VisItem
	menu, err = s.MainMenu(ctx)
	if err != nil {
		return entity.SiteSettings{}, err
	}
	postsPerPage, _ := strconv.Atoi(s.repository.Setting(ctx, "posts_per_page", strconv.Itoa(defaultPostsPerPage)))
	if postsPerPage == 0 {
		postsPerPage = defaultPostsPerPage
	}
	return entity.SiteSettings{
		SiteName:        s.repository.Setting(ctx, "site_name", defaultSiteName),
		SiteIconURL:     s.repository.Setting(ctx, "site_icon_url", ""),
		SiteDescription: s.repository.SettingI18n(ctx, "site_seo", language, ""),
		PostsPerPage:    postsPerPage,
		HomeHero:        hero,
		HomeSections:    sections,
		MainMenu:        menu,
	}, nil
}

// Hero returns the home hero block.
func (s *Service) Hero(ctx context.Context) (entity.HomeHero, error) {
	var ctas []entity.HeroCta
	var err error
	ctas, err = s.repository.ListHeroCtas(ctx)
	if err != nil {
		return entity.HomeHero{}, fmt.Errorf("get home hero: %w", err)
	}
	if ctas == nil {
		ctas = []entity.HeroCta{}
	}
	return entity.HomeHero{
		TitleVisible:    strings.EqualFold(s.repository.Setting(ctx, "hero_title_visible", "true"), "true"),
		SubtitleVisible: strings.EqualFold(s.repository.Setting(ctx, "hero_subtitle_visible", "true"), "true"),
		Ctas:            ctas,
	}, nil
}

// HomeSections returns home sections with normalized item limits.
func (s *Service) HomeSections(ctx context.Context) ([]entity.VisItem, error) {
	var rows []entity.HomeSectionRecord
	var err error
	rows, err = s.repository.ListHomeSections(ctx)
	if err != nil {
		return nil, fmt.Errorf("get home sections: %w", err)
	}
	out := []entity.VisItem{}
	for _, row := range rows {
		limit := normalizeLimit(row.ItemLimit)
		out = append(out, entity.VisItem{ID: row.Key, Visible: row.Visible, ItemLimit: &limit})
	}
	return out, nil
}

// MainMenu returns the main menu. Item limits stay unset.
func (s *Service) MainMenu(ctx context.Context) ([]entity.VisItem, error) {
	var rows []entity.VisItem
	var err error
	rows, err = s.repository.ListMainMenu(ctx)
	if err != nil {
		return nil, fmt.Errorf("get main menu: %w", err)
	}
	if rows == nil {
		return []entity.VisItem{}, nil
	}
	return rows, nil
}
