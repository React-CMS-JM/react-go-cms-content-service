package post

import (
	"context"
	"fmt"
	"strings"

	"react-go-cms-content-service/internal/domain/apperror"
	"react-go-cms-content-service/internal/domain/entity"
)

// List returns one page of localized posts.
func (s *Service) List(ctx context.Context, contentType, status, lang string, page, size int) (entity.Page[entity.Post], error) {
	language := normalizeLang(lang)
	if page < minPageIndex {
		page = minPageIndex
	}
	if size < minPageSize {
		size = minPageSize
	}
	if size > maxPageSize {
		size = maxPageSize
	}
	filter := ListFilter{Page: page, Size: size}
	if strings.TrimSpace(contentType) != "" {
		slug := strings.ToLower(strings.TrimSpace(contentType))
		if slug == courseSlug || !ownedContentType(slug) {
			return entity.Page[entity.Post]{}, fmt.Errorf("list posts: %w", apperror.Invalid("Unsupported content type: "+slug))
		}
		filter.TypeSlug = slug
	}
	if strings.TrimSpace(status) != "" {
		filter.Status = strings.TrimSpace(status)
	}
	var records []entity.PostRecord
	var total int64
	var err error
	records, total, err = s.repository.List(ctx, filter)
	if err != nil {
		return entity.Page[entity.Post]{}, fmt.Errorf("list posts: %w", err)
	}
	var posts []entity.Post
	posts, err = s.localize(ctx, records, language, true)
	if err != nil {
		return entity.Page[entity.Post]{}, fmt.Errorf("list posts: %w", err)
	}
	return entity.Page[entity.Post]{Items: posts, Page: page, Size: size, Total: total}, nil
}

// Get returns one owned post in the requested language.
func (s *Service) Get(ctx context.Context, id, lang string) (entity.Post, error) {
	var record entity.PostRecord
	var err error
	record, err = s.requireOwned(ctx, id)
	if err != nil {
		return entity.Post{}, err
	}
	var posts []entity.Post
	posts, err = s.localize(ctx, []entity.PostRecord{record}, normalizeLang(lang), true)
	if err != nil || len(posts) == 0 {
		if err != nil {
			return entity.Post{}, fmt.Errorf("get post: %w", err)
		}
		return entity.Post{}, nil
	}
	return posts[0], nil
}

// GetBySlug returns one owned post for a slug, optionally constrained by content type.
func (s *Service) GetBySlug(ctx context.Context, slug, contentType, lang string) (entity.Post, error) {
	language := normalizeLang(lang)
	var postID string
	var err error
	postID, err = s.repository.FindIDBySlug(ctx, slug, language)
	if err != nil {
		return entity.Post{}, fmt.Errorf("get post by slug: %w", err)
	}
	var record entity.PostRecord
	record, err = s.requireOwned(ctx, postID)
	if err != nil {
		return entity.Post{}, err
	}
	if strings.TrimSpace(contentType) != "" {
		var typeSlug string
		typeSlug, err = s.repository.ContentTypeSlug(ctx, record.ContentTypeID)
		if err != nil || !strings.EqualFold(typeSlug, contentType) {
			return entity.Post{}, fmt.Errorf("get post by slug: %w", apperror.NotFound("Post not found for slug/type: "+slug+"/"+contentType))
		}
	}
	var posts []entity.Post
	posts, err = s.localize(ctx, []entity.PostRecord{record}, language, true)
	if err != nil || len(posts) == 0 {
		if err != nil {
			return entity.Post{}, fmt.Errorf("get post by slug: %w", err)
		}
		return entity.Post{}, nil
	}
	return posts[0], nil
}

// Metadata returns the metadata rows for an owned post.
func (s *Service) Metadata(ctx context.Context, id string) ([]entity.Metadata, error) {
	_, err := s.requireOwned(ctx, id)
	if err != nil {
		return nil, err
	}
	var rows []entity.Metadata
	rows, err = s.repository.ListMetadata(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list post metadata: %w", err)
	}
	return rows, nil
}

func (s *Service) requireOwned(ctx context.Context, id string) (entity.PostRecord, error) {
	var record entity.PostRecord
	var err error
	record, err = s.repository.FindByID(ctx, id)
	if err != nil {
		return entity.PostRecord{}, fmt.Errorf("require post: %w", err)
	}
	var slug string
	slug, err = s.repository.ContentTypeSlug(ctx, record.ContentTypeID)
	if err != nil || slug == courseSlug || !ownedContentType(slug) {
		return entity.PostRecord{}, fmt.Errorf("require post: %w", apperror.NotFound("Post not found: "+id))
	}
	return record, nil
}

func (s *Service) localize(ctx context.Context, records []entity.PostRecord, lang string, withMetadata bool) ([]entity.Post, error) {
	out := make([]entity.Post, 0, len(records))
	if len(records) == 0 {
		return out, nil
	}
	var detailed []entity.PostRecord
	var err error
	detailed, err = s.repository.LoadDetails(ctx, records, withMetadata)
	if err != nil {
		return nil, err
	}
	for _, record := range detailed {
		item := entity.Post{
			ID:               record.ID,
			AuthorID:         record.AuthorID,
			ContentTypeID:    record.ContentTypeID,
			FeaturedImageURL: record.FeaturedImageURL,
			AccessLevel:      record.AccessLevel,
			Status:           record.Status,
			ViewCount:        record.ViewCount,
			PublishedAt:      record.PublishedAt,
			CreatedAt:        record.CreatedAt,
			UpdatedAt:        record.UpdatedAt,
			CategoryIDs:      record.CategoryIDs,
			TagIDs:           record.TagIDs,
			Metadata:         []entity.Metadata{},
		}
		if item.CategoryIDs == nil {
			item.CategoryIDs = []int{}
		}
		if item.TagIDs == nil {
			item.TagIDs = []int{}
		}
		translation := pickTranslation(record.Translations, lang)
		if translation == nil {
			item.LanguageCode = lang
			item.Title = ""
			item.Slug = ""
			item.Content = ""
			item.Excerpt = emptyString()
			item.MetaTitle = emptyString()
			item.MetaDescription = emptyString()
		} else {
			item.LanguageCode = translation.LanguageCode
			item.Title = translation.Title
			item.Slug = translation.Slug
			item.Content = translation.Content
			item.Excerpt = translation.Excerpt
			item.MetaTitle = translation.MetaTitle
			item.MetaDescription = translation.MetaDescription
		}
		if withMetadata && record.Metadata != nil {
			item.Metadata = record.Metadata
		}
		out = append(out, item)
	}
	return out, nil
}

func pickTranslation(rows []entity.PostTranslation, lang string) *entity.PostTranslation {
	if len(rows) == 0 {
		return nil
	}
	for index := range rows {
		if rows[index].LanguageCode == lang {
			return &rows[index]
		}
	}
	return &rows[0]
}

func (s *Service) refreshUsage(ctx context.Context, categoryIDs, tagIDs []int) {
	_ = s.categories.RefreshUsageCounts(ctx, categoryIDs)
	_ = s.tags.RefreshUsageCounts(ctx, tagIDs)
}
