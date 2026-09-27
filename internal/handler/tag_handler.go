package handler

import (
	"net/http"

	"react-go-cms-content-service/internal/domain/apperror"
	"react-go-cms-content-service/internal/domain/entity"
	"react-go-cms-content-service/internal/infrastructure/httpx"
)

func (h *Handler) listTags(w http.ResponseWriter, r *http.Request) {
	var rows []entity.Taxonomy
	var err error
	rows, err = h.tags.ListPopular(r.Context(), r.URL.Query().Get("lang"))
	writeTaxonomy(w, rows, err)
}

func (h *Handler) searchTags(w http.ResponseWriter, r *http.Request) {
	var rows []entity.Taxonomy
	var err error
	rows, err = h.tags.Search(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("lang"), httpx.QueryInt(r, "limit", defaultLimit))
	writeTaxonomy(w, rows, err)
}

func (h *Handler) tagsByIDs(w http.ResponseWriter, r *http.Request) {
	var rows []entity.Taxonomy
	var err error
	rows, err = h.tags.ByIDs(r.Context(), parseIDs(r.URL.Query().Get("ids")), r.URL.Query().Get("lang"))
	writeTaxonomy(w, rows, err)
}

func (h *Handler) adminTags(w http.ResponseWriter, r *http.Request) {
	var page entity.Page[entity.Taxonomy]
	var err error
	page, err = h.tags.Admin(r.Context(), r.URL.Query().Get("lang"), httpx.QueryInt(r, "page", defaultPage), httpx.QueryInt(r, "size", defaultAdminSize), r.URL.Query().Get("q"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toTaxonomyPage(page))
}

func (h *Handler) createTag(w http.ResponseWriter, r *http.Request) {
	var body TaxonomyRequest
	var err error
	err = httpx.DecodeJSONLenient(r, &body)
	if err != nil {
		writeServiceError(w, apperror.Invalid(invalidJSON))
		return
	}
	var item entity.Taxonomy
	item, err = h.tags.Create(r.Context(), toTaxonomyInput(body))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toTaxonomyResponse(item))
}

func (h *Handler) updateTag(w http.ResponseWriter, r *http.Request) {
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
	item, err = h.tags.Update(r.Context(), id, toTaxonomyInput(body))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toTaxonomyResponse(item))
}

func (h *Handler) deleteTag(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r)
	if !ok {
		return
	}
	var err error
	err = h.tags.Delete(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteNoContent(w)
}
