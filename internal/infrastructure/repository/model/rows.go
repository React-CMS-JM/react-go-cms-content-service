// Package model holds database row shapes for the content service.
package model

import "database/sql"

// PostRow is one posts row.
type PostRow struct {
	ID               string         `db:"id"`
	AuthorID         string         `db:"author_id"`
	ContentTypeID    int            `db:"content_type_id"`
	FeaturedImageURL sql.NullString `db:"featured_image_url"`
	AccessLevel      string         `db:"access_level"`
	Status           string         `db:"status"`
	ViewCount        int            `db:"view_count"`
	PublishedAt      sql.NullTime   `db:"published_at"`
	CreatedAt        sql.NullTime   `db:"created_at"`
	UpdatedAt        sql.NullTime   `db:"updated_at"`
}

// PostI18nRow is one post_i18n row.
type PostI18nRow struct {
	PostID          string         `db:"post_id"`
	LanguageCode    string         `db:"language_code"`
	Title           string         `db:"title"`
	Slug            string         `db:"slug"`
	Content         string         `db:"content"`
	Excerpt         sql.NullString `db:"excerpt"`
	MetaTitle       sql.NullString `db:"meta_title"`
	MetaDescription sql.NullString `db:"meta_description"`
}

// MetadataRow is one post_metadata row.
type MetadataRow struct {
	ID        string         `db:"id"`
	PostID    string         `db:"post_id"`
	MetaKey   string         `db:"meta_key"`
	MetaValue sql.NullString `db:"meta_value"`
}

// TaxonomyRow is one categories or tags row.
type TaxonomyRow struct {
	ID            int            `db:"id"`
	DBDescription sql.NullString `db:"db_description"`
	UsageCount    int64          `db:"usage_count"`
}

// TaxonomyI18nRow is one category_i18n or tag_i18n row.
type TaxonomyI18nRow struct {
	OwnerID      int    `db:"owner_id"`
	LanguageCode string `db:"language_code"`
	Name         string `db:"name"`
	Slug         string `db:"slug"`
}

// ContentTypeRow is one content_types row.
type ContentTypeRow struct {
	ID          int            `db:"id"`
	Name        string         `db:"name"`
	Slug        string         `db:"slug"`
	Description sql.NullString `db:"description"`
}

// CommentRow is one comments row.
type CommentRow struct {
	ID              string         `db:"id"`
	PostID          string         `db:"post_id"`
	UserID          string         `db:"user_id"`
	ParentCommentID sql.NullString `db:"parent_comment_id"`
	Status          string         `db:"status"`
	CreatedAt       sql.NullTime   `db:"created_at"`
	UpdatedAt       sql.NullTime   `db:"updated_at"`
}

// CommentI18nRow is one comment_i18n row.
type CommentI18nRow struct {
	CommentID    string `db:"comment_id"`
	LanguageCode string `db:"language_code"`
	Content      string `db:"content"`
	IsOriginal   int    `db:"is_original"`
}

// HeroCtaRow is one hero_ctas row.
type HeroCtaRow struct {
	ID              string `db:"id"`
	Href            string `db:"href"`
	TextColor       string `db:"text_color"`
	BackgroundColor string `db:"background_color"`
	IsVisible       int    `db:"is_visible"`
}

// HomeSectionRow is one home_sections row.
type HomeSectionRow struct {
	SectionKey string `db:"section_key"`
	IsVisible  int    `db:"is_visible"`
	ItemLimit  int    `db:"item_limit"`
}

// NavItemRow is one nav_items row.
type NavItemRow struct {
	ItemKey   string `db:"item_key"`
	IsVisible int    `db:"is_visible"`
}

// UIStringKeyRow is one param_ui_strings row.
type UIStringKeyRow struct {
	StringKey   string         `db:"string_key"`
	UIComponent sql.NullString `db:"ui_component"`
}

// RecentPostRow is one dashboard recent-post row.
type RecentPostRow struct {
	ID            string       `db:"id"`
	AuthorID      string       `db:"author_id"`
	ContentTypeID int          `db:"content_type_id"`
	Status        string       `db:"status"`
	UpdatedAt     sql.NullTime `db:"updated_at"`
}
