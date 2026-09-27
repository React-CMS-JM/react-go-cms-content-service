package handler

import (
	"net/http"

	"react-go-cms-content-service/internal/domain/entity"
	"react-go-cms-content-service/internal/infrastructure/httpx"
)

// ContentTypeResponse is one content type.
type ContentTypeResponse struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description *string `json:"description"`
}

func (h *Handler) contentTypesList(w http.ResponseWriter, r *http.Request) {
	var rows []entity.ContentType
	var err error
	rows, err = h.contentTypes.List(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]ContentTypeResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, ContentTypeResponse{ID: row.ID, Name: row.Name, Slug: row.Slug, Description: row.Description})
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
