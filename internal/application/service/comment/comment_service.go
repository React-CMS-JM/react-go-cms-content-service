// Package comment applies comment use cases.
package comment

import (
	"strings"

	"react-go-cms-content-service/internal/domain/entity"
)

const (
	defaultLanguage = "en"
	defaultStatus   = "approved"
)

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
