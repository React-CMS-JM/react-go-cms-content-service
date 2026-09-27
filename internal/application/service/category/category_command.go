package category

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"react-go-cms-content-service/internal/domain/apperror"
	"react-go-cms-content-service/internal/domain/entity"
)

// Create stores a category and its first translation.
func (s *Service) Create(ctx context.Context, req entity.TaxonomyInput) (entity.Taxonomy, error) {
	if req.Name == nil || strings.TrimSpace(*req.Name) == "" {
		return entity.Taxonomy{}, fmt.Errorf("create category: %w", apperror.Invalid("name is required"))
	}
	description := *req.Name
	if req.DBDescription != nil {
		description = *req.DBDescription
	}
	lang := defaultLanguage
	if req.LanguageCode != nil {
		lang = normalizeLang(*req.LanguageCode)
	}
	slug := slugify(*req.Name)
	if req.Slug != nil && strings.TrimSpace(*req.Slug) != "" {
		slug = strings.TrimSpace(*req.Slug)
	}
	var id int
	var err error
	id, err = s.repository.Insert(ctx, description, lang, *req.Name, slug)
	if err != nil {
		return entity.Taxonomy{}, fmt.Errorf("create category: %w", err)
	}
	s.repository.ClearPopular()
	return s.reload(ctx, id, lang, "create category")
}

// Update changes a category and one translation.
func (s *Service) Update(ctx context.Context, id int, req entity.TaxonomyInput) (entity.Taxonomy, error) {
	var exists bool
	var err error
	exists, err = s.repository.Exists(ctx, id)
	if err != nil {
		return entity.Taxonomy{}, fmt.Errorf("update category: %w", err)
	}
	if !exists {
		return entity.Taxonomy{}, fmt.Errorf("update category: %w", apperror.NotFound(notFoundLabel+" not found: "+strconv.Itoa(id)))
	}
	if req.DBDescription != nil {
		err = s.repository.UpdateDescription(ctx, id, *req.DBDescription)
		if err != nil {
			return entity.Taxonomy{}, fmt.Errorf("update category: %w", err)
		}
	}
	lang := defaultLanguage
	if req.LanguageCode != nil {
		lang = normalizeLang(*req.LanguageCode)
	}
	var translationID int
	var found bool
	translationID, found, err = s.repository.FindTranslation(ctx, id, lang)
	if err != nil {
		return entity.Taxonomy{}, fmt.Errorf("update category: %w", err)
	}
	if !found {
		if req.Name == nil || strings.TrimSpace(*req.Name) == "" {
			return entity.Taxonomy{}, fmt.Errorf("update category: %w", apperror.Invalid("name is required when creating a translation"))
		}
		slug := slugify(*req.Name)
		if req.Slug != nil && strings.TrimSpace(*req.Slug) != "" {
			slug = strings.TrimSpace(*req.Slug)
		}
		err = s.repository.InsertTranslation(ctx, id, lang, strings.TrimSpace(*req.Name), slug)
		if err != nil {
			return entity.Taxonomy{}, fmt.Errorf("update category: %w", err)
		}
	} else {
		err = s.updateExistingTranslation(ctx, translationID, req)
		if err != nil {
			return entity.Taxonomy{}, fmt.Errorf("update category: %w", err)
		}
	}
	s.repository.ClearPopular()
	return s.reload(ctx, id, lang, "update category")
}

func (s *Service) updateExistingTranslation(ctx context.Context, translationID int, req entity.TaxonomyInput) error {
	if req.Name != nil {
		var err error
		err = s.repository.UpdateTranslationName(ctx, translationID, *req.Name)
		if err != nil {
			return err
		}
	}
	if req.Slug != nil {
		return s.repository.UpdateTranslationSlug(ctx, translationID, *req.Slug)
	}
	if req.Name != nil {
		return s.repository.UpdateTranslationSlug(ctx, translationID, slugify(*req.Name))
	}
	return nil
}

// Delete removes a category, its translations, and post links.
func (s *Service) Delete(ctx context.Context, id int) error {
	var deleted bool
	var err error
	deleted, err = s.repository.DeleteRow(ctx, id)
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	if !deleted {
		return fmt.Errorf("delete category: %w", apperror.NotFound(notFoundLabel+" not found: "+strconv.Itoa(id)))
	}
	_ = s.repository.DeleteTranslations(ctx, id)
	_ = s.repository.DeleteLinks(ctx, id)
	s.repository.ClearPopular()
	return nil
}

// RefreshUsageCounts recalculates usage for the given categories and clears the popular cache.
func (s *Service) RefreshUsageCounts(ctx context.Context, ids []int) error {
	return s.repository.RefreshUsageCounts(ctx, ids)
}

func (s *Service) reload(ctx context.Context, id int, lang, action string) (entity.Taxonomy, error) {
	var items []entity.Taxonomy
	var err error
	items, err = s.repository.ByIDs(ctx, []int{id}, lang)
	if err != nil || len(items) == 0 {
		if err != nil {
			return entity.Taxonomy{}, fmt.Errorf("%s: %w", action, err)
		}
		return entity.Taxonomy{}, nil
	}
	return items[0], nil
}
