package handler

import (
	"net/http"

	"react-go-cms-content-service/internal/domain/apperror"
	"react-go-cms-content-service/internal/domain/entity"
	"react-go-cms-content-service/internal/infrastructure/httpx"
)

// TaxonomyResponse is a localized category or tag.
type TaxonomyResponse struct {
	ID            int     `json:"id"`
	DBDescription *string `json:"dbDescription"`
	LanguageCode  string  `json:"languageCode"`
	Name          string  `json:"name"`
	Slug          string  `json:"slug"`
	UsageCount    int64   `json:"usageCount"`
}

// TaxonomyPage is a JSON page of categories or tags.
type TaxonomyPage struct {
	Items []TaxonomyResponse `json:"items"`
	Page  int                `json:"page"`
	Size  int                `json:"size"`
	Total int64              `json:"total"`
}

// TaxonomyRequest is a category or tag write body.
type TaxonomyRequest struct {
	LanguageCode  *string `json:"languageCode"`
	Name          *string `json:"name"`
	Slug          *string `json:"slug"`
	DBDescription *string `json:"dbDescription"`
}

func (h *Handler) listCategories(w http.ResponseWriter, r *http.Request) {
	var rows []entity.Taxonomy
	var err error
	rows, err = h.categories.ListPopular(r.Context(), r.URL.Query().Get("lang"))
	writeTaxonomy(w, rows, err)
}

func (h *Handler) searchCategories(w http.ResponseWriter, r *http.Request) {
	var rows []entity.Taxonomy
	var err error
	rows, err = h.categories.Search(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("lang"), httpx.QueryInt(r, "limit", defaultLimit))
	writeTaxonomy(w, rows, err)
}

func (h *Handler) categoriesByIDs(w http.ResponseWriter, r *http.Request) {
	var rows []entity.Taxonomy
	var err error
	rows, err = h.categories.ByIDs(r.Context(), parseIDs(r.URL.Query().Get("ids")), r.URL.Query().Get("lang"))
	writeTaxonomy(w, rows, err)
}

func (h *Handler) adminCategories(w http.ResponseWriter, r *http.Request) {
	var page entity.Page[entity.Taxonomy]
	var err error
	page, err = h.categories.Admin(r.Context(), r.URL.Query().Get("lang"), httpx.QueryInt(r, "page", defaultPage), httpx.QueryInt(r, "size", defaultAdminSize), r.URL.Query().Get("q"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toTaxonomyPage(page))
}

func (h *Handler) createCategory(w http.ResponseWriter, r *http.Request) {
	var body TaxonomyRequest
	var err error
	err = httpx.DecodeJSONLenient(r, &body)
	if err != nil {
		writeServiceError(w, apperror.Invalid(invalidJSON))
		return
	}
	var item entity.Taxonomy
	item, err = h.categories.Create(r.Context(), toTaxonomyInput(body))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toTaxonomyResponse(item))
}

func (h *Handler) updateCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r)
	if !ok {
		return
	}
	var body TaxonomyRequest
	var err error
	err = httpx.DecodeJSONLenient(r, &body)
	if err != nil {
		writeServiceError(w, apperror.Invalid(invalidJSON))
		return
	}
	var item entity.Taxonomy
	item, err = h.categories.Update(r.Context(), id, toTaxonomyInput(body))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toTaxonomyResponse(item))
}

func (h *Handler) deleteCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r)
	if !ok {
		return
	}
	var err error
	err = h.categories.Delete(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteNoContent(w)
}

func writeTaxonomy(w http.ResponseWriter, rows []entity.Taxonomy, err error) {
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toTaxonomyResponses(rows))
}

func toTaxonomyPage(page entity.Page[entity.Taxonomy]) TaxonomyPage {
	return TaxonomyPage{Items: toTaxonomyResponses(page.Items), Page: page.Page, Size: page.Size, Total: page.Total}
}

func toTaxonomyResponses(rows []entity.Taxonomy) []TaxonomyResponse {
	if rows == nil {
		return []TaxonomyResponse{}
	}
	out := make([]TaxonomyResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, toTaxonomyResponse(row))
	}
	return out
}

func toTaxonomyResponse(item entity.Taxonomy) TaxonomyResponse {
	return TaxonomyResponse{
		ID:            item.ID,
		DBDescription: item.DBDescription,
		LanguageCode:  item.LanguageCode,
		Name:          item.Name,
		Slug:          item.Slug,
		UsageCount:    item.UsageCount,
	}
}

func toTaxonomyInput(body TaxonomyRequest) entity.TaxonomyInput {
	return entity.TaxonomyInput{
		LanguageCode:  body.LanguageCode,
		Name:          body.Name,
		Slug:          body.Slug,
		DBDescription: body.DBDescription,
	}
}
