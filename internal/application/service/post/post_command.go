package post

import (
	"context"
	"fmt"
	"strings"
	"time"

	"react-go-cms-content-service/internal/domain/apperror"
	"react-go-cms-content-service/internal/domain/entity"
)

const publishedAtLayout = "2006-01-02 15:04:05"

// Create stores a post and returns it in the written language.
func (s *Service) Create(ctx context.Context, req entity.PostCreate, fallbackAuthor string) (entity.Post, error) {
	if strings.TrimSpace(req.Title) == "" {
		return entity.Post{}, fmt.Errorf("create post: %w", apperror.Invalid("title is required"))
	}
	if req.Content == nil {
		return entity.Post{}, fmt.Errorf("create post: %w", apperror.Invalid("content is required"))
	}
	var typeID int
	var err error
	typeID, err = s.resolveType(ctx, req.ContentTypeID, req.ContentTypeSlug)
	if err != nil {
		return entity.Post{}, err
	}
	author := fallbackAuthor
	if req.AuthorID != nil && strings.TrimSpace(*req.AuthorID) != "" {
		author = strings.TrimSpace(*req.AuthorID)
	}
	if strings.TrimSpace(author) == "" {
		return entity.Post{}, fmt.Errorf("create post: %w", apperror.Invalid("authorId is required"))
	}
	lang := defaultLanguage
	if req.LanguageCode != nil {
		lang = normalizeLang(*req.LanguageCode)
	}
	slug := slugify(req.Title)
	if req.Slug != nil && strings.TrimSpace(*req.Slug) != "" {
		slug = strings.TrimSpace(*req.Slug)
	}
	access := accessPublic
	if req.AccessLevel != nil {
		access = *req.AccessLevel
	}
	status := statusDraft
	if req.Status != nil {
		status = *req.Status
	}
	var published *string
	if status == statusPublished {
		formatted := time.Now().UTC().Format(publishedAtLayout)
		published = &formatted
	}
	var id string
	var categoryIDs []int
	var tagIDs []int
	id, categoryIDs, tagIDs, err = s.repository.Insert(ctx, InsertInput{
		AuthorID:         author,
		ContentTypeID:    typeID,
		FeaturedImageURL: req.FeaturedImageURL,
		AccessLevel:      access,
		Status:           status,
		PublishedAt:      published,
		LanguageCode:     lang,
		Title:            req.Title,
		Slug:             slug,
		Content:          *req.Content,
		Excerpt:          req.Excerpt,
		MetaTitle:        req.MetaTitle,
		MetaDescription:  req.MetaDescription,
		CategoryIDs:      req.CategoryIDs,
		TagIDs:           req.TagIDs,
	})
	if err != nil {
		return entity.Post{}, fmt.Errorf("create post: %w", err)
	}
	s.refreshUsage(ctx, categoryIDs, tagIDs)
	return s.Get(ctx, id, lang)
}

// Update stores a partial post change and reloads it.
func (s *Service) Update(ctx context.Context, id string, req entity.PostUpdate) (entity.Post, error) {
	_, err := s.requireOwned(ctx, id)
	if err != nil {
		return entity.Post{}, err
	}
	input := UpdateInput{
		FeaturedImageURL: req.FeaturedImageURL,
		AccessLevel:      req.AccessLevel,
		Status:           req.Status,
		CategoryIDs:      req.CategoryIDs,
		TagIDs:           req.TagIDs,
		Title:            req.Title,
		Slug:             req.Slug,
		Content:          req.Content,
		Excerpt:          req.Excerpt,
		MetaTitle:        req.MetaTitle,
		MetaDescription:  req.MetaDescription,
	}
	if req.ContentTypeID != nil {
		var typeID int
		typeID, err = s.resolveType(ctx, req.ContentTypeID, nil)
		if err != nil {
			return entity.Post{}, err
		}
		input.ContentTypeID = &typeID
	}
	lang := defaultLanguage
	if req.LanguageCode != nil {
		lang = normalizeLang(*req.LanguageCode)
	}
	input.LanguageCode = lang
	var categoryIDs []int
	var tagIDs []int
	categoryIDs, tagIDs, err = s.repository.Update(ctx, id, input)
	if err != nil {
		return entity.Post{}, fmt.Errorf("update post: %w", err)
	}
	s.refreshUsage(ctx, categoryIDs, tagIDs)
	return s.Get(ctx, id, lang)
}

// PatchStatus changes publication state and reloads the post in English.
func (s *Service) PatchStatus(ctx context.Context, id, status string) (entity.Post, error) {
	if strings.TrimSpace(status) == "" {
		return entity.Post{}, fmt.Errorf("patch post status: %w", apperror.Invalid("status is required"))
	}
	_, err := s.requireOwned(ctx, id)
	if err != nil {
		return entity.Post{}, err
	}
	err = s.repository.ApplyStatus(ctx, id, strings.TrimSpace(status))
	if err != nil {
		return entity.Post{}, fmt.Errorf("patch post status: %w", err)
	}
	return s.Get(ctx, id, defaultLanguage)
}

// IncrementView adds one view and reloads the post in English.
func (s *Service) IncrementView(ctx context.Context, id string) (entity.Post, error) {
	_, err := s.requireOwned(ctx, id)
	if err != nil {
		return entity.Post{}, err
	}
	err = s.repository.IncrementView(ctx, id)
	if err != nil {
		return entity.Post{}, fmt.Errorf("increment post view: %w", err)
	}
	return s.Get(ctx, id, defaultLanguage)
}

// Delete removes an owned post and refreshes linked taxonomy counts.
func (s *Service) Delete(ctx context.Context, id string) error {
	_, err := s.requireOwned(ctx, id)
	if err != nil {
		return err
	}
	var categoryIDs []int
	var tagIDs []int
	categoryIDs, tagIDs, err = s.repository.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("delete post: %w", err)
	}
	s.refreshUsage(ctx, categoryIDs, tagIDs)
	return nil
}

// ReplaceMetadata replaces metadata rows and returns the stored set.
func (s *Service) ReplaceMetadata(ctx context.Context, id string, entries []entity.MetadataInput) ([]entity.Metadata, error) {
	_, err := s.requireOwned(ctx, id)
	if err != nil {
		return nil, err
	}
	err = s.repository.ReplaceMetadata(ctx, id, entries)
	if err != nil {
		return nil, fmt.Errorf("replace post metadata: %w", err)
	}
	return s.Metadata(ctx, id)
}

func (s *Service) resolveType(ctx context.Context, id *int, slug *string) (int, error) {
	var typeID int
	var typeSlug string
	var err error
	typeID, typeSlug, err = s.repository.FindContentType(ctx, id, slug)
	if err != nil {
		return 0, fmt.Errorf("resolve content type: %w", err)
	}
	if typeSlug == courseSlug || !ownedContentType(typeSlug) {
		return 0, fmt.Errorf("resolve content type: %w", apperror.Invalid("Content type not managed by content-service: "+typeSlug))
	}
	return typeID, nil
}
