package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"react-go-cms-content-service/internal/domain/apperror"
	"react-go-cms-content-service/internal/domain/entity"
	"react-go-cms-content-service/internal/infrastructure/httpx"
)

// UIStringResponse is one localized UI string.
type UIStringResponse struct {
	StringKey    string    `json:"stringKey"`
	UIComponent  *string   `json:"uiComponent"`
	LanguageCode string    `json:"languageCode"`
	StringValue  string    `json:"stringValue"`
	UpdatedAt    LocalJSON `json:"updatedAt"`
}

type upsertUIBody struct {
	LanguageCode *string `json:"languageCode"`
	StringValue  *string `json:"stringValue"`
}

type patchUIBody struct {
	Items []struct {
		StringKey    string `json:"stringKey"`
		LanguageCode string `json:"languageCode"`
		StringValue  string `json:"stringValue"`
	} `json:"items"`
}

func (h *Handler) listUI(w http.ResponseWriter, r *http.Request) {
	var rows []entity.UIString
	var err error
	rows, err = h.uiStrings.List(r.Context(), r.URL.Query().Get("lang"), r.URL.Query().Get("component"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toUIStringResponses(rows))
}

func (h *Handler) upsertUI(w http.ResponseWriter, r *http.Request) {
	var body upsertUIBody
	var err error
	err = httpx.DecodeJSONLenient(r, &body)
	if err != nil {
		writeServiceError(w, apperror.Invalid(invalidJSON))
		return
	}
	lang := ""
	if body.LanguageCode != nil {
		lang = *body.LanguageCode
	}
	var item entity.UIString
	item, err = h.uiStrings.Upsert(r.Context(), chi.URLParam(r, "stringKey"), lang, body.StringValue)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toUIStringResponse(item))
}

func (h *Handler) patchUI(w http.ResponseWriter, r *http.Request) {
	var body patchUIBody
	var err error
	err = httpx.DecodeJSONLenient(r, &body)
	if err != nil || body.Items == nil {
		writeServiceError(w, apperror.Invalid("items are required"))
		return
	}
	items := make([]entity.UIStringWrite, 0, len(body.Items))
	for _, item := range body.Items {
		items = append(items, entity.UIStringWrite{StringKey: item.StringKey, LanguageCode: item.LanguageCode, StringValue: item.StringValue})
	}
	var rows []entity.UIString
	rows, err = h.uiStrings.Patch(r.Context(), items)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toUIStringResponses(rows))
}

func toUIStringResponses(rows []entity.UIString) []UIStringResponse {
	if rows == nil {
		return []UIStringResponse{}
	}
	out := make([]UIStringResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, toUIStringResponse(row))
	}
	return out
}

func toUIStringResponse(item entity.UIString) UIStringResponse {
	return UIStringResponse{
		StringKey:    item.StringKey,
		UIComponent:  item.UIComponent,
		LanguageCode: item.LanguageCode,
		StringValue:  item.StringValue,
		UpdatedAt:    toLocal(item.UpdatedAt),
	}
}
