package settings

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"react-go-cms-content-service/internal/domain/entity"
)

// Put stores site settings and any nested hero, sections, or menu included in the body.
func (s *Service) Put(ctx context.Context, body entity.SiteSettingsInput, lang string) (entity.SiteSettings, error) {
	language := normalizeLang(lang)
	var err error
	err = s.repository.UpsertSetting(ctx, "site_name", body.SiteName)
	if err != nil {
		return entity.SiteSettings{}, fmt.Errorf("put settings: %w", err)
	}
	icon := ""
	if body.SiteIconURL != nil {
		icon = *body.SiteIconURL
	}
	err = s.repository.UpsertSetting(ctx, "site_icon_url", icon)
	if err != nil {
		return entity.SiteSettings{}, fmt.Errorf("put settings: %w", err)
	}
	postsPerPage := body.PostsPerPage
	if postsPerPage <= 0 {
		postsPerPage = defaultPostsPerPage
	}
	err = s.repository.UpsertSetting(ctx, "posts_per_page", strconv.Itoa(postsPerPage))
	if err != nil {
		return entity.SiteSettings{}, fmt.Errorf("put settings: %w", err)
	}
	if body.SiteDescription != nil {
		err = s.repository.UpsertSettingI18n(ctx, "site_seo", language, *body.SiteDescription)
		if err != nil {
			return entity.SiteSettings{}, fmt.Errorf("put settings: %w", err)
		}
	}
	if body.HomeHero != nil {
		_, err = s.PutHero(ctx, *body.HomeHero)
		if err != nil {
			return entity.SiteSettings{}, err
		}
	}
	if body.HomeSections != nil {
		_, err = s.PutHomeSections(ctx, *body.HomeSections)
		if err != nil {
			return entity.SiteSettings{}, err
		}
	}
	if body.MainMenu != nil {
		_, err = s.PutMainMenu(ctx, *body.MainMenu)
		if err != nil {
			return entity.SiteSettings{}, err
		}
	}
	return s.Get(ctx, language)
}

// PutHero replaces hero visibility flags and call-to-action rows.
func (s *Service) PutHero(ctx context.Context, hero entity.HomeHeroInput) (entity.HomeHero, error) {
	title := true
	if hero.TitleVisible != nil {
		title = *hero.TitleVisible
	}
	subtitle := true
	if hero.SubtitleVisible != nil {
		subtitle = *hero.SubtitleVisible
	}
	var err error
	err = s.repository.UpsertSetting(ctx, "hero_title_visible", strconv.FormatBool(title))
	if err != nil {
		return entity.HomeHero{}, fmt.Errorf("put home hero: %w", err)
	}
	err = s.repository.UpsertSetting(ctx, "hero_subtitle_visible", strconv.FormatBool(subtitle))
	if err != nil {
		return entity.HomeHero{}, fmt.Errorf("put home hero: %w", err)
	}
	writes := make([]entity.HeroCtaWrite, 0, len(hero.Ctas))
	for _, cta := range hero.Ctas {
		href := heroHref
		if cta.Href != nil {
			href = *cta.Href
		}
		textColor := heroTextColor
		if cta.TextColor != nil {
			textColor = *cta.TextColor
		}
		background := heroBackground
		if cta.BackgroundColor != nil {
			background = *cta.BackgroundColor
		}
		visible := true
		if cta.Visible != nil && !*cta.Visible {
			visible = false
		}
		writes = append(writes, entity.HeroCtaWrite{
			ID:              strings.TrimSpace(cta.ID),
			Href:            href,
			TextColor:       textColor,
			BackgroundColor: background,
			Visible:         visible,
			Labels:          cta.Labels,
		})
	}
	err = s.repository.ReplaceHeroCtas(ctx, writes)
	if err != nil {
		return entity.HomeHero{}, fmt.Errorf("put home hero: %w", err)
	}
	return s.Hero(ctx)
}

// PutHomeSections upserts each non-blank section and returns the stored list.
func (s *Service) PutHomeSections(ctx context.Context, items []entity.VisItemInput) ([]entity.VisItem, error) {
	for index, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			continue
		}
		visible := true
		if item.Visible != nil {
			visible = *item.Visible
		}
		limit := normalizeLimitPtr(item.ItemLimit, defaultSectionLimit(item.ID))
		var err error
		err = s.repository.SaveHomeSection(ctx, index, item.ID, visible, limit)
		if err != nil {
			return nil, fmt.Errorf("put home sections: %w", err)
		}
	}
	return s.HomeSections(ctx)
}

// PutMainMenu upserts each non-blank menu item and returns the stored list.
func (s *Service) PutMainMenu(ctx context.Context, items []entity.VisItemInput) ([]entity.VisItem, error) {
	for index, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			continue
		}
		visible := true
		if item.Visible != nil {
			visible = *item.Visible
		}
		var err error
		err = s.repository.SaveMenuItem(ctx, index, item.ID, visible)
		if err != nil {
			return nil, fmt.Errorf("put main menu: %w", err)
		}
	}
	return s.MainMenu(ctx)
}
