package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"react-go-cms-content-service/internal/domain/apperror"
	"react-go-cms-content-service/internal/domain/entity"
	"react-go-cms-content-service/internal/infrastructure/httpx"
)

// CommentResponse is one comment payload.
type CommentResponse struct {
	ID                            string    `json:"id"`
	PostID                        string    `json:"postId"`
	UserID                        string    `json:"userId"`
	ParentCommentID               *string   `json:"parentCommentId"`
	Content                       string    `json:"content"`
	LanguageCode                  string    `json:"languageCode"`
	AvailableTranslationLanguages []string  `json:"availableTranslationLanguages"`
	Status                        string    `json:"status"`
	CreatedAt                     LocalJSON `json:"createdAt"`
	UpdatedAt                     LocalJSON `json:"updatedAt"`
}

// CommentTranslationResponse is one comment translation.
type CommentTranslationResponse struct {
	CommentID    string `json:"commentId"`
	LanguageCode string `json:"languageCode"`
	Content      string `json:"content"`
	IsOriginal   bool   `json:"isOriginal"`
}

// CreateCommentRequest is the create-comment body.
type CreateCommentRequest struct {
	PostID          string  `json:"postId"`
	UserID          *string `json:"userId"`
	ParentCommentID *string `json:"parentCommentId"`
	Content         string  `json:"content"`
	LanguageCode    *string `json:"languageCode"`
	Status          *string `json:"status"`
}

type updateCommentBody struct {
	Content      string  `json:"content"`
	LanguageCode *string `json:"languageCode"`
}

func (h *Handler) listComments(w http.ResponseWriter, r *http.Request) {
	var rows []entity.Comment
	var err error
	rows, err = h.comments.List(r.Context(), r.URL.Query().Get("postId"), r.URL.Query().Get("status"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toCommentResponses(rows))
}

func (h *Handler) createComment(w http.ResponseWriter, r *http.Request) {
	var body CreateCommentRequest
	var err error
	err = httpx.DecodeJSONLenient(r, &body)
	if err != nil {
		writeServiceError(w, apperror.Invalid(invalidJSON))
		return
	}
	var item entity.Comment
	item, err = h.comments.Create(r.Context(), entity.CommentCreate{
		PostID:          body.PostID,
		UserID:          body.UserID,
		ParentCommentID: body.ParentCommentID,
		Content:         body.Content,
		LanguageCode:    body.LanguageCode,
		Status:          body.Status,
	}, subject(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toCommentResponse(item))
}

func (h *Handler) updateComment(w http.ResponseWriter, r *http.Request) {
	var body updateCommentBody
	var err error
	err = httpx.DecodeJSONLenient(r, &body)
	if err != nil {
		writeServiceError(w, apperror.Invalid(invalidJSON))
		return
	}
	var item entity.Comment
	item, err = h.comments.Update(r.Context(), chi.URLParam(r, "id"), body.Content, body.LanguageCode)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toCommentResponse(item))
}

func (h *Handler) getCommentTranslation(w http.ResponseWriter, r *http.Request) {
	var item entity.CommentTranslation
	var err error
	item, err = h.comments.Translation(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "lang"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toCommentTranslation(item))
}

func (h *Handler) putCommentTranslation(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	_ = httpx.DecodeJSONLenient(r, &body)
	var item entity.CommentTranslation
	var err error
	item, err = h.comments.UpsertTranslation(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "lang"), body["content"])
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toCommentTranslation(item))
}

func (h *Handler) patchComment(w http.ResponseWriter, r *http.Request) {
	var body statusBody
	_ = httpx.DecodeJSONLenient(r, &body)
	status := ""
	if body.Status != nil {
		status = *body.Status
	}
	var item entity.Comment
	var err error
	item, err = h.comments.PatchStatus(r.Context(), chi.URLParam(r, "id"), status)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toCommentResponse(item))
}

func (h *Handler) deleteComment(w http.ResponseWriter, r *http.Request) {
	var err error
	err = h.comments.Delete(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteNoContent(w)
}

func toCommentResponses(rows []entity.Comment) []CommentResponse {
	if rows == nil {
		return []CommentResponse{}
	}
	out := make([]CommentResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, toCommentResponse(row))
	}
	return out
}

func toCommentResponse(item entity.Comment) CommentResponse {
	languages := item.AvailableTranslationLanguages
	if languages == nil {
		languages = []string{}
	}
	return CommentResponse{
		ID:                            item.ID,
		PostID:                        item.PostID,
		UserID:                        item.UserID,
		ParentCommentID:               item.ParentCommentID,
		Content:                       item.Content,
		LanguageCode:                  item.LanguageCode,
		AvailableTranslationLanguages: languages,
		Status:                        item.Status,
		CreatedAt:                     toLocal(item.CreatedAt),
		UpdatedAt:                     toLocal(item.UpdatedAt),
	}
}

func toCommentTranslation(item entity.CommentTranslation) CommentTranslationResponse {
	return CommentTranslationResponse{
		CommentID:    item.CommentID,
		LanguageCode: item.LanguageCode,
		Content:      item.Content,
		IsOriginal:   item.IsOriginal,
	}
}
