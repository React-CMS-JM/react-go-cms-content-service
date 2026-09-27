package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"react-go-cms-content-service/internal/domain/apperror"
	"react-go-cms-content-service/internal/domain/entity"
	"react-go-cms-content-service/internal/infrastructure/httpx"
)

// PostPage is a JSON page of posts.
type PostPage struct {
	Items []PostResponse `json:"items"`
	Page  int            `json:"page"`
	Size  int            `json:"size"`
	Total int64          `json:"total"`
}

// MetadataResponse is one post metadata object.
type MetadataResponse struct {
	ID        string  `json:"id"`
	PostID    string  `json:"postId"`
	MetaKey   string  `json:"metaKey"`
	MetaValue *string `json:"metaValue"`
}

// PostResponse is the localized post payload.
type PostResponse struct {
	ID               string             `json:"id"`
	AuthorID         string             `json:"authorId"`
	ContentTypeID    int                `json:"contentTypeId"`
	FeaturedImageURL *string            `json:"featuredImageUrl"`
	AccessLevel      string             `json:"accessLevel"`
	Status           string             `json:"status"`
	ViewCount        int                `json:"viewCount"`
	PublishedAt      LocalJSON          `json:"publishedAt"`
	CreatedAt        LocalJSON          `json:"createdAt"`
	UpdatedAt        LocalJSON          `json:"updatedAt"`
	CategoryIDs      []int              `json:"categoryIds"`
	TagIDs           []int              `json:"tagIds"`
	LanguageCode     string             `json:"languageCode"`
	Title            string             `json:"title"`
	Slug             string             `json:"slug"`
	Content          string             `json:"content"`
	Excerpt          *string            `json:"excerpt"`
	MetaTitle        *string            `json:"metaTitle"`
	MetaDescription  *string            `json:"metaDescription"`
	Metadata         []MetadataResponse `json:"metadata"`
}

// CreatePostRequest is the create-post body.
type CreatePostRequest struct {
	AuthorID         *string `json:"authorId"`
	ContentTypeID    *int    `json:"contentTypeId"`
	ContentTypeSlug  *string `json:"contentTypeSlug"`
	FeaturedImageURL *string `json:"featuredImageUrl"`
	AccessLevel      *string `json:"accessLevel"`
	Status           *string `json:"status"`
	CategoryIDs      []int   `json:"categoryIds"`
	TagIDs           []int   `json:"tagIds"`
	LanguageCode     *string `json:"languageCode"`
	Title            string  `json:"title"`
	Slug             *string `json:"slug"`
	Content          *string `json:"content"`
	Excerpt          *string `json:"excerpt"`
	MetaTitle        *string `json:"metaTitle"`
	MetaDescription  *string `json:"metaDescription"`
}

// UpdatePostRequest is the update-post body.
type UpdatePostRequest struct {
	ContentTypeID    *int    `json:"contentTypeId"`
	FeaturedImageURL *string `json:"featuredImageUrl"`
	AccessLevel      *string `json:"accessLevel"`
	Status           *string `json:"status"`
	CategoryIDs      *[]int  `json:"categoryIds"`
	TagIDs           *[]int  `json:"tagIds"`
	LanguageCode     *string `json:"languageCode"`
	Title            *string `json:"title"`
	Slug             *string `json:"slug"`
	Content          *string `json:"content"`
	Excerpt          *string `json:"excerpt"`
	MetaTitle        *string `json:"metaTitle"`
	MetaDescription  *string `json:"metaDescription"`
}

// MetadataEntryRequest is one metadata write entry.
type MetadataEntryRequest struct {
	MetaKey   string  `json:"metaKey"`
	MetaValue *string `json:"metaValue"`
}

type metadataBody struct {
	Entries []MetadataEntryRequest `json:"entries"`
}

type statusBody struct {
	Status *string `json:"status"`
}

func (h *Handler) listPosts(w http.ResponseWriter, r *http.Request) {
	var page entity.Page[entity.Post]
	var err error
	page, err = h.posts.List(r.Context(), r.URL.Query().Get("type"), r.URL.Query().Get("status"), r.URL.Query().Get("lang"), httpx.QueryInt(r, "page", defaultPage), httpx.QueryInt(r, "size", defaultPageSize))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toPostPage(page))
}

func (h *Handler) getPost(w http.ResponseWriter, r *http.Request) {
	var item entity.Post
	var err error
	item, err = h.posts.Get(r.Context(), chi.URLParam(r, "id"), r.URL.Query().Get("lang"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toPostResponse(item))
}

func (h *Handler) postBySlug(w http.ResponseWriter, r *http.Request) {
	var item entity.Post
	var err error
	item, err = h.posts.GetBySlug(r.Context(), chi.URLParam(r, "slug"), r.URL.Query().Get("type"), r.URL.Query().Get("lang"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toPostResponse(item))
}

func (h *Handler) createPost(w http.ResponseWriter, r *http.Request) {
	var body CreatePostRequest
	var err error
	err = httpx.DecodeJSONLenient(r, &body)
	if err != nil {
		writeServiceError(w, apperror.Invalid(invalidJSON))
		return
	}
	var item entity.Post
	item, err = h.posts.Create(r.Context(), toPostCreate(body), subject(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toPostResponse(item))
}

func (h *Handler) updatePost(w http.ResponseWriter, r *http.Request) {
	var body UpdatePostRequest
	var err error
	err = httpx.DecodeJSONLenient(r, &body)
	if err != nil {
		writeServiceError(w, apperror.Invalid(invalidJSON))
		return
	}
	var item entity.Post
	item, err = h.posts.Update(r.Context(), chi.URLParam(r, "id"), toPostUpdate(body))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toPostResponse(item))
}

func (h *Handler) patchPostStatus(w http.ResponseWriter, r *http.Request) {
	var body statusBody
	_ = httpx.DecodeJSONLenient(r, &body)
	status := ""
	if body.Status != nil {
		status = *body.Status
	}
	var item entity.Post
	var err error
	item, err = h.posts.PatchStatus(r.Context(), chi.URLParam(r, "id"), status)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toPostResponse(item))
}

func (h *Handler) viewPost(w http.ResponseWriter, r *http.Request) {
	var item entity.Post
	var err error
	item, err = h.posts.IncrementView(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toPostResponse(item))
}

func (h *Handler) deletePost(w http.ResponseWriter, r *http.Request) {
	var err error
	err = h.posts.Delete(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteNoContent(w)
}

func (h *Handler) getMetadata(w http.ResponseWriter, r *http.Request) {
	var rows []entity.Metadata
	var err error
	rows, err = h.posts.Metadata(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toMetadataResponses(rows))
}

func (h *Handler) putMetadata(w http.ResponseWriter, r *http.Request) {
	var body metadataBody
	var err error
	err = httpx.DecodeJSONLenient(r, &body)
	if err != nil {
		writeServiceError(w, apperror.Invalid(invalidJSON))
		return
	}
	var rows []entity.Metadata
	rows, err = h.posts.ReplaceMetadata(r.Context(), chi.URLParam(r, "id"), toMetadataInputs(body.Entries))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toMetadataResponses(rows))
}

func toPostPage(page entity.Page[entity.Post]) PostPage {
	items := make([]PostResponse, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, toPostResponse(item))
	}
	return PostPage{Items: items, Page: page.Page, Size: page.Size, Total: page.Total}
}

func toPostResponse(item entity.Post) PostResponse {
	return PostResponse{
		ID:               item.ID,
		AuthorID:         item.AuthorID,
		ContentTypeID:    item.ContentTypeID,
		FeaturedImageURL: item.FeaturedImageURL,
		AccessLevel:      item.AccessLevel,
		Status:           item.Status,
		ViewCount:        item.ViewCount,
		PublishedAt:      toLocal(item.PublishedAt),
		CreatedAt:        toLocal(item.CreatedAt),
		UpdatedAt:        toLocal(item.UpdatedAt),
		CategoryIDs:      item.CategoryIDs,
		TagIDs:           item.TagIDs,
		LanguageCode:     item.LanguageCode,
		Title:            item.Title,
		Slug:             item.Slug,
		Content:          item.Content,
		Excerpt:          item.Excerpt,
		MetaTitle:        item.MetaTitle,
		MetaDescription:  item.MetaDescription,
		Metadata:         toMetadataResponses(item.Metadata),
	}
}

func toMetadataResponses(rows []entity.Metadata) []MetadataResponse {
	if rows == nil {
		return []MetadataResponse{}
	}
	out := make([]MetadataResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, MetadataResponse{ID: row.ID, PostID: row.PostID, MetaKey: row.MetaKey, MetaValue: row.MetaValue})
	}
	return out
}

func toMetadataInputs(rows []MetadataEntryRequest) []entity.MetadataInput {
	out := make([]entity.MetadataInput, 0, len(rows))
	for _, row := range rows {
		out = append(out, entity.MetadataInput{MetaKey: row.MetaKey, MetaValue: row.MetaValue})
	}
	return out
}

func toPostCreate(body CreatePostRequest) entity.PostCreate {
	return entity.PostCreate{
		AuthorID:         body.AuthorID,
		ContentTypeID:    body.ContentTypeID,
		ContentTypeSlug:  body.ContentTypeSlug,
		FeaturedImageURL: body.FeaturedImageURL,
		AccessLevel:      body.AccessLevel,
		Status:           body.Status,
		CategoryIDs:      body.CategoryIDs,
		TagIDs:           body.TagIDs,
		LanguageCode:     body.LanguageCode,
		Title:            body.Title,
		Slug:             body.Slug,
		Content:          body.Content,
		Excerpt:          body.Excerpt,
		MetaTitle:        body.MetaTitle,
		MetaDescription:  body.MetaDescription,
	}
}

func toPostUpdate(body UpdatePostRequest) entity.PostUpdate {
	return entity.PostUpdate{
		ContentTypeID:    body.ContentTypeID,
		FeaturedImageURL: body.FeaturedImageURL,
		AccessLevel:      body.AccessLevel,
		Status:           body.Status,
		CategoryIDs:      body.CategoryIDs,
		TagIDs:           body.TagIDs,
		LanguageCode:     body.LanguageCode,
		Title:            body.Title,
		Slug:             body.Slug,
		Content:          body.Content,
		Excerpt:          body.Excerpt,
		MetaTitle:        body.MetaTitle,
		MetaDescription:  body.MetaDescription,
	}
}
