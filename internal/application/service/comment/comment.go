// Package comment applies comment use cases.
package comment

import (
	"context"
	"strings"

	"react-go-cms-content-service/internal/domain/entity"
)

const (
	defaultLanguage = "en"
	defaultStatus   = "approved"
)

// Repository is the comment persistence port.
type Repository interface {
	List(ctx context.Context, postID, status string) ([]entity.CommentRecord, error)
	PostExists(ctx context.Context, postID string) (bool, error)
	Insert(ctx context.Context, postID, userID string, parent *string, status, lang, content string) (string, error)
	Require(ctx context.Context, id string) error
	ReplaceContent(ctx context.Context, id, lang, content string) error
	ReadTranslation(ctx context.Context, id, lang string) (languageCode, content string, original bool, found bool, err error)
	FindTranslation(ctx context.Context, id, lang string) (rowID int, original bool, found bool, err error)
	InsertTranslation(ctx context.Context, id, lang, content string) error
	UpdateTranslation(ctx context.Context, rowID int, content string) error
	PatchStatus(ctx context.Context, id, status string) (bool, error)
	Delete(ctx context.Context, id string) error
	Find(ctx context.Context, id string) (entity.CommentRecord, error)
}

// Service coordinates comment commands and queries.
type Service struct {
	repository Repository
}

// New builds a comment service.
func New(repository Repository) *Service {
	return &Service{repository: repository}
}

func normalizeLang(lang string) string {
	lang = strings.TrimSpace(strings.ToLower(lang))
	if lang == "" {
		return defaultLanguage
	}
	return lang
}

func toComment(record entity.CommentRecord) entity.Comment {
	comment := entity.Comment{
		ID:                            record.ID,
		PostID:                        record.PostID,
		UserID:                        record.UserID,
		ParentCommentID:               record.ParentCommentID,
		Status:                        record.Status,
		CreatedAt:                     record.CreatedAt,
		UpdatedAt:                     record.UpdatedAt,
		AvailableTranslationLanguages: []string{},
		Content:                       "",
		LanguageCode:                  defaultLanguage,
	}
	var original *entity.CommentText
	for index := range record.Translations {
		if record.Translations[index].Original && original == nil {
			original = &record.Translations[index]
		} else if !record.Translations[index].Original {
			comment.AvailableTranslationLanguages = append(comment.AvailableTranslationLanguages, record.Translations[index].LanguageCode)
		}
	}
	if original == nil && len(record.Translations) > 0 {
		original = &record.Translations[0]
	}
	if original != nil {
		comment.Content = original.Content
		comment.LanguageCode = original.LanguageCode
	}
	return comment
}
