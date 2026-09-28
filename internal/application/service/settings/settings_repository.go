package settings

import (
	"context"

	"react-go-cms-content-service/internal/domain/entity"
)

// Repository is the settings persistence port.
type Repository interface {
	Setting(ctx context.Context, key, fallback string) string
	SettingI18n(ctx context.Context, key, lang, fallback string) string
	UpsertSetting(ctx context.Context, key, value string) error
	UpsertSettingI18n(ctx context.Context, key, lang, value string) error
	ListHeroCtas(ctx context.Context) ([]entity.HeroCta, error)
	ReplaceHeroCtas(ctx context.Context, ctas []entity.HeroCtaWrite) error
	ListHomeSections(ctx context.Context) ([]entity.HomeSectionRecord, error)
	SaveHomeSection(ctx context.Context, index int, id string, visible bool, limit int) error
	ListMainMenu(ctx context.Context) ([]entity.VisItem, error)
	SaveMenuItem(ctx context.Context, index int, id string, visible bool) error
}
