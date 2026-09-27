package entity

import "time"

// CommentText is one active comment translation row.
type CommentText struct {
	LanguageCode string
	Content      string
	Original     bool
}

// CommentRecord is a stored comment plus its active translations.
type CommentRecord struct {
	ID              string
	PostID          string
	UserID          string
	ParentCommentID *string
	Status          string
	CreatedAt       *time.Time
	UpdatedAt       *time.Time
	Translations    []CommentText
}

// Comment is a comment shaped for clients.
type Comment struct {
	ID                            string
	PostID                        string
	UserID                        string
	ParentCommentID               *string
	Content                       string
	LanguageCode                  string
	AvailableTranslationLanguages []string
	Status                        string
	CreatedAt                     *time.Time
	UpdatedAt                     *time.Time
}

// CommentTranslation is one comment translation.
type CommentTranslation struct {
	CommentID    string
	LanguageCode string
	Content      string
	IsOriginal   bool
}

// CommentCreate is the create-comment command.
type CommentCreate struct {
	PostID          string
	UserID          *string
	ParentCommentID *string
	Content         string
	LanguageCode    *string
	Status          *string
}
