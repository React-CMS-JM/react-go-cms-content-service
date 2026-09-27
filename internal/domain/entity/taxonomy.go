package entity

// Taxonomy is a localized category or tag.
type Taxonomy struct {
	ID            int
	DBDescription *string
	LanguageCode  string
	Name          string
	Slug          string
	UsageCount    int64
}

// TaxonomyInput is a category or tag write payload.
type TaxonomyInput struct {
	LanguageCode  *string
	Name          *string
	Slug          *string
	DBDescription *string
}
