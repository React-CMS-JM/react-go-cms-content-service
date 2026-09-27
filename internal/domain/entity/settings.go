package entity

// HeroCta is one home-hero call to action.
type HeroCta struct {
	ID              string
	Visible         bool
	Href            string
	TextColor       string
	BackgroundColor string
	Labels          map[string]string
}

// HomeHero is the home hero block.
type HomeHero struct {
	TitleVisible    bool
	SubtitleVisible bool
	Ctas            []HeroCta
}

// VisItem is a home section or main-menu entry.
type VisItem struct {
	ID        string
	Visible   bool
	ItemLimit *int
}

// SiteSettings is the public site settings document.
type SiteSettings struct {
	SiteName        string
	SiteIconURL     string
	SiteDescription string
	PostsPerPage    int
	HomeHero        HomeHero
	HomeSections    []VisItem
	MainMenu        []VisItem
}

// HeroCtaInput is a hero call-to-action write payload.
type HeroCtaInput struct {
	ID              string
	Visible         *bool
	Href            *string
	TextColor       *string
	BackgroundColor *string
	Labels          map[string]string
}

// HomeHeroInput is a home-hero write payload.
type HomeHeroInput struct {
	TitleVisible    *bool
	SubtitleVisible *bool
	Ctas            []HeroCtaInput
}

// VisItemInput is a home-section or menu write payload.
type VisItemInput struct {
	ID        string
	Visible   *bool
	ItemLimit *int
}

// SiteSettingsInput is a site-settings write payload.
type SiteSettingsInput struct {
	SiteName        string
	SiteIconURL     *string
	SiteDescription *string
	PostsPerPage    int
	HomeHero        *HomeHeroInput
	HomeSections    *[]VisItemInput
	MainMenu        *[]VisItemInput
}

// HomeSectionRecord is a home_sections row before item-limit normalization.
type HomeSectionRecord struct {
	Key       string
	Visible   bool
	ItemLimit int
}

// HeroCtaWrite is a hero call to action ready to insert.
type HeroCtaWrite struct {
	ID              string
	Href            string
	TextColor       string
	BackgroundColor string
	Visible         bool
	Labels          map[string]string
}
