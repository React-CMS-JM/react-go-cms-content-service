package comment

import (
	"context"
	"fmt"

	"react-go-cms-content-service/internal/domain/apperror"
	"react-go-cms-content-service/internal/domain/entity"
)

// List returns comments for an optional post and status filter.
func (s *Service) List(ctx context.Context, postID, status string) ([]entity.Comment, error) {
	var records []entity.CommentRecord
	var err error
	records, err = s.repository.List(ctx, postID, status)
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	out := make([]entity.Comment, 0, len(records))
	for _, record := range records {
		out = append(out, toComment(record))
	}
	return out, nil
}

// Translation returns one active translation.
func (s *Service) Translation(ctx context.Context, id, lang string) (entity.CommentTranslation, error) {
	var err error
	err = s.repository.Require(ctx, id)
	if err != nil {
		return entity.CommentTranslation{}, fmt.Errorf("get comment translation: %w", err)
	}
	language := normalizeLang(lang)
	var storedLang string
	var content string
	var original bool
	var found bool
	storedLang, content, original, found, err = s.repository.ReadTranslation(ctx, id, language)
	if err != nil {
		return entity.CommentTranslation{}, fmt.Errorf("get comment translation: %w", err)
	}
	if !found {
		return entity.CommentTranslation{}, fmt.Errorf("get comment translation: %w", apperror.NotFound("Translation not found for comment "+id+" lang="+language))
	}
	return entity.CommentTranslation{CommentID: id, LanguageCode: storedLang, Content: content, IsOriginal: original}, nil
}

func (s *Service) byID(ctx context.Context, id string) (entity.Comment, error) {
	var record entity.CommentRecord
	var err error
	record, err = s.repository.Find(ctx, id)
	if err != nil {
		return entity.Comment{}, fmt.Errorf("load comment: %w", err)
	}
	return toComment(record), nil
}
