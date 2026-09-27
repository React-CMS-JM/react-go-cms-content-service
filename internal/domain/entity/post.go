package entity

import "time"

// Metadata is one post metadata row.
type Metadata struct {
	ID        string
	PostID    string
	MetaKey   string
	MetaValue *string
}

// MetadataInput is a metadata write payload.
type MetadataInput struct {
	MetaKey   string
	MetaValue *string
}

// PostTranslation is one localized post text.
type PostTranslation struct {
	LanguageCode    string
	Title           string
	Slug            string
	Content         string
	Excerpt         *string
	MetaTitle       *string
	MetaDescription *string
}

// PostRecord is a stored post plus every translation, before language selection.
type PostRecord struct {
	ID               string
	AuthorID         string
	ContentTypeID    int
	FeaturedImageURL *string
	AccessLevel      string
	Status           string
	ViewCount        int
	PublishedAt      *time.Time
	CreatedAt        *time.Time
	UpdatedAt        *time.Time
	CategoryIDs      []int
	TagIDs           []int
	Translations     []PostTranslation
	Metadata         []Metadata
}

// Post is a post localized to one language.
type Post struct {
	ID               string
	AuthorID         string
	ContentTypeID    int
	FeaturedImageURL *string
	AccessLevel      string
	Status           string
	ViewCount        int
	PublishedAt      *time.Time
	CreatedAt        *time.Time
	UpdatedAt        *time.Time
	CategoryIDs      []int
	TagIDs           []int
	LanguageCode     string
	Title            string
	Slug             string
	Content          string
	Excerpt          *string
	MetaTitle        *string
	MetaDescription  *string
	Metadata         []Metadata
}

// PostCreate is the create-post command.
type PostCreate struct {
	AuthorID         *string
	ContentTypeID    *int
	ContentTypeSlug  *string
	FeaturedImageURL *string
	AccessLevel      *string
	Status           *string
	CategoryIDs      []int
	TagIDs           []int
	LanguageCode     *string
	Title            string
	Slug             *string
	Content          *string
	Excerpt          *string
	MetaTitle        *string
	MetaDescription  *string
}

// PostUpdate is the partial update-post command.
type PostUpdate struct {
	ContentTypeID    *int
	FeaturedImageURL *string
	AccessLevel      *string
	Status           *string
	CategoryIDs      *[]int
	TagIDs           *[]int
	LanguageCode     *string
	Title            *string
	Slug             *string
	Content          *string
	Excerpt          *string
	MetaTitle        *string
	MetaDescription  *string
}
