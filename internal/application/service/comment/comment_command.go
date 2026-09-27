package comment

import (
	"context"
	"fmt"
	"strings"

	"react-go-cms-content-service/internal/domain/apperror"
	"react-go-cms-content-service/internal/domain/entity"
)

// Create stores a comment and its original translation.
func (s *Service) Create(ctx context.Context, req entity.CommentCreate, fallbackUser string) (entity.Comment, error) {
	if strings.TrimSpace(req.PostID) == "" {
		return entity.Comment{}, fmt.Errorf("create comment: %w", apperror.Invalid("postId is required"))
	}
	if strings.TrimSpace(req.Content) == "" {
		return entity.Comment{}, fmt.Errorf("create comment: %w", apperror.Invalid("content is required"))
	}
	var exists bool
	var err error
	exists, err = s.repository.PostExists(ctx, req.PostID)
	if err != nil {
		return entity.Comment{}, fmt.Errorf("create comment: %w", err)
	}
	if !exists {
		return entity.Comment{}, fmt.Errorf("create comment: %w", apperror.NotFound("Post not found: "+req.PostID))
	}
	userID := fallbackUser
	if req.UserID != nil && strings.TrimSpace(*req.UserID) != "" {
		userID = strings.TrimSpace(*req.UserID)
	}
	if strings.TrimSpace(userID) == "" {
		return entity.Comment{}, fmt.Errorf("create comment: %w", apperror.Invalid("userId is required"))
	}
	lang := defaultLanguage
	if req.LanguageCode != nil {
		lang = normalizeLang(*req.LanguageCode)
	}
	status := defaultStatus
	if req.Status != nil && strings.TrimSpace(*req.Status) != "" {
		status = *req.Status
	}
	var id string
	id, err = s.repository.Insert(ctx, req.PostID, userID, req.ParentCommentID, status, lang, req.Content)
	if err != nil {
		return entity.Comment{}, fmt.Errorf("create comment: %w", err)
	}
	return s.byID(ctx, id)
}

// Update replaces the active comment text with a new original translation.
func (s *Service) Update(ctx context.Context, id, content string, lang *string) (entity.Comment, error) {
	if strings.TrimSpace(content) == "" {
		return entity.Comment{}, fmt.Errorf("update comment: %w", apperror.Invalid("content is required"))
	}
	var err error
	err = s.repository.Require(ctx, id)
	if err != nil {
		return entity.Comment{}, fmt.Errorf("update comment: %w", err)
	}
	language := defaultLanguage
	if lang != nil {
		language = normalizeLang(*lang)
	}
	err = s.repository.ReplaceContent(ctx, id, language, content)
	if err != nil {
		return entity.Comment{}, fmt.Errorf("update comment: %w", err)
	}
	return s.byID(ctx, id)
}

// UpsertTranslation writes one translation and returns it.
func (s *Service) UpsertTranslation(ctx context.Context, id, lang, content string) (entity.CommentTranslation, error) {
	if strings.TrimSpace(content) == "" {
		return entity.CommentTranslation{}, fmt.Errorf("upsert comment translation: %w", apperror.Invalid("content is required"))
	}
	var err error
	err = s.repository.Require(ctx, id)
	if err != nil {
		return entity.CommentTranslation{}, fmt.Errorf("upsert comment translation: %w", err)
	}
	language := normalizeLang(lang)
	var rowID int
	var original bool
	var found bool
	rowID, original, found, err = s.repository.FindTranslation(ctx, id, language)
	if err != nil {
		return entity.CommentTranslation{}, fmt.Errorf("upsert comment translation: %w", err)
	}
	if !found {
		err = s.repository.InsertTranslation(ctx, id, language, content)
		if err != nil {
			return entity.CommentTranslation{}, fmt.Errorf("upsert comment translation: %w", err)
		}
		return entity.CommentTranslation{CommentID: id, LanguageCode: language, Content: content, IsOriginal: false}, nil
	}
	err = s.repository.UpdateTranslation(ctx, rowID, content)
	if err != nil {
		return entity.CommentTranslation{}, fmt.Errorf("upsert comment translation: %w", err)
	}
	return entity.CommentTranslation{CommentID: id, LanguageCode: language, Content: content, IsOriginal: original}, nil
}

// PatchStatus changes a comment status.
func (s *Service) PatchStatus(ctx context.Context, id, status string) (entity.Comment, error) {
	if strings.TrimSpace(status) == "" {
		return entity.Comment{}, fmt.Errorf("patch comment status: %w", apperror.Invalid("status is required"))
	}
	var updated bool
	var err error
	updated, err = s.repository.PatchStatus(ctx, id, strings.TrimSpace(status))
	if err != nil {
		return entity.Comment{}, fmt.Errorf("patch comment status: %w", err)
	}
	if !updated {
		return entity.Comment{}, fmt.Errorf("patch comment status: %w", apperror.NotFound("Comment not found: "+id))
	}
	return s.byID(ctx, id)
}

// Delete removes a comment and its translations.
func (s *Service) Delete(ctx context.Context, id string) error {
	var err error
	err = s.repository.Require(ctx, id)
	if err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}
	err = s.repository.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}
	return nil
}
