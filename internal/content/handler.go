package content

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"react-go-cms-content-service/internal/platform"
)

type Handler struct {
	store  *Store
	secret string
	issuer string
}

func NewHandler(store *Store, secret, issuer string) *Handler {
	return &Handler{store: store, secret: secret, issuer: issuer}
}

func (h *Handler) Register(mux chi.Router) {
	mux.Get("/api/content/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("content-service-ok"))
	})

	mux.Get("/api/admin/dashboard", h.auth(h.dashboard))
	mux.Get("/api/content-types", h.contentTypes)

	mux.Get("/api/posts/by-slug/{slug}", h.postBySlug)
	mux.Get("/api/posts/{id}/metadata", h.getMetadata)
	mux.Put("/api/posts/{id}/metadata", h.auth(h.putMetadata))
	mux.Patch("/api/posts/{id}/status", h.auth(h.patchPostStatus))
	mux.Post("/api/posts/{id}/view", h.auth(h.view))
	mux.Get("/api/posts/{id}", h.getPost)
	mux.Put("/api/posts/{id}", h.auth(h.updatePost))
	mux.Delete("/api/posts/{id}", h.auth(h.deletePost))
	mux.Get("/api/posts", h.listPosts)
	mux.Post("/api/posts", h.auth(h.createPost))

	mux.Get("/api/categories/search", h.searchCategories)
	mux.Get("/api/categories/by-ids", h.categoriesByIDs)
	mux.Get("/api/categories/admin", h.adminCategories)
	mux.Put("/api/categories/{id}", h.auth(h.updateCategory))
	mux.Delete("/api/categories/{id}", h.auth(h.deleteCategory))
	mux.Get("/api/categories", h.listCategories)
	mux.Post("/api/categories", h.auth(h.createCategory))

	mux.Get("/api/tags/search", h.searchTags)
	mux.Get("/api/tags/by-ids", h.tagsByIDs)
	mux.Get("/api/tags/admin", h.adminTags)
	mux.Put("/api/tags/{id}", h.auth(h.updateTag))
	mux.Delete("/api/tags/{id}", h.auth(h.deleteTag))
	mux.Get("/api/tags", h.listTags)
	mux.Post("/api/tags", h.auth(h.createTag))

	mux.Get("/api/comments/{id}/translations/{lang}", h.getTranslation)
	mux.Put("/api/comments/{id}/translations/{lang}", h.auth(h.putTranslation))
	mux.Patch("/api/comments/{id}/status", h.auth(h.patchComment))
	mux.Put("/api/comments/{id}", h.auth(h.updateComment))
	mux.Delete("/api/comments/{id}", h.auth(h.deleteComment))
	mux.Get("/api/comments", h.listComments)
	mux.Post("/api/comments", h.auth(h.createComment))

	mux.Get("/api/settings/home-hero", h.getHero)
	mux.Put("/api/settings/home-hero", h.auth(h.putHero))
	mux.Get("/api/settings/home-sections", h.getSections)
	mux.Put("/api/settings/home-sections", h.auth(h.putSections))
	mux.Get("/api/settings/main-menu", h.getMenu)
	mux.Put("/api/settings/main-menu", h.auth(h.putMenu))
	mux.Get("/api/settings", h.getSettings)
	mux.Put("/api/settings", h.auth(h.putSettings))

	mux.Put("/api/ui-strings/{stringKey}", h.auth(h.upsertUI))
	mux.Patch("/api/ui-strings", h.auth(h.patchUI))
	mux.Get("/api/ui-strings", h.listUI)
}

func (h *Handler) contentTypes(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.db.QueryContext(r.Context(), `SELECT id, name, slug, description FROM content_types ORDER BY id`)
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	defer rows.Close()
	type dto struct {
		ID          int     `json:"id"`
		Name        string  `json:"name"`
		Slug        string  `json:"slug"`
		Description *string `json:"description"`
	}
	out := []dto{}
	for rows.Next() {
		var d dto
		var desc sql.NullString
		if err := rows.Scan(&d.ID, &d.Name, &d.Slug, &desc); err != nil {
			platform.WriteError(w, err, "error")
			return
		}
		d.Description = platform.StrPtr(desc)
		out = append(out, d)
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	d, err := h.store.Dashboard(r.Context(), r.URL.Query().Get("lang"), platform.QueryInt(r, "recentLimit", 6), r.Header.Get("Authorization"))
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, d)
}

func (h *Handler) listPosts(w http.ResponseWriter, r *http.Request) {
	page, err := h.store.ListPosts(r.Context(), r.URL.Query().Get("type"), r.URL.Query().Get("status"), r.URL.Query().Get("lang"), platform.QueryInt(r, "page", 0), platform.QueryInt(r, "size", 20))
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, page)
}

func (h *Handler) getPost(w http.ResponseWriter, r *http.Request) {
	p, err := h.store.GetPost(r.Context(), chi.URLParam(r, "id"), r.URL.Query().Get("lang"))
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, p)
}

func (h *Handler) postBySlug(w http.ResponseWriter, r *http.Request) {
	p, err := h.store.GetPostBySlug(r.Context(), chi.URLParam(r, "slug"), r.URL.Query().Get("type"), r.URL.Query().Get("lang"))
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, p)
}

func (h *Handler) createPost(w http.ResponseWriter, r *http.Request) {
	var body CreatePost
	if err := platform.DecodeJSONLenient(r, &body); err != nil {
		platform.WriteError(w, bad("invalid JSON body"), "error")
		return
	}
	p, err := h.store.CreatePost(r.Context(), body, subject(r))
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusCreated, p)
}

func (h *Handler) updatePost(w http.ResponseWriter, r *http.Request) {
	var body UpdatePost
	if err := platform.DecodeJSONLenient(r, &body); err != nil {
		platform.WriteError(w, bad("invalid JSON body"), "error")
		return
	}
	p, err := h.store.UpdatePost(r.Context(), chi.URLParam(r, "id"), body)
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, p)
}

func (h *Handler) patchPostStatus(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Status *string `json:"status"`
	}
	_ = platform.DecodeJSONLenient(r, &body)
	status := ""
	if body.Status != nil {
		status = *body.Status
	}
	p, err := h.store.PatchPostStatus(r.Context(), chi.URLParam(r, "id"), status)
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, p)
}

func (h *Handler) view(w http.ResponseWriter, r *http.Request) {
	p, err := h.store.IncrementView(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, p)
}

func (h *Handler) deletePost(w http.ResponseWriter, r *http.Request) {
	if err := h.store.DeletePost(r.Context(), chi.URLParam(r, "id")); err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteNoContent(w)
}

func (h *Handler) getMetadata(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.GetMetadata(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, rows)
}

func (h *Handler) putMetadata(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Entries []MetaEntry `json:"entries"`
	}
	if err := platform.DecodeJSONLenient(r, &body); err != nil {
		platform.WriteError(w, bad("invalid JSON body"), "error")
		return
	}
	rows, err := h.store.PutMetadata(r.Context(), chi.URLParam(r, "id"), body.Entries)
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, rows)
}

func (h *Handler) listCategories(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.ListPopularCategories(r.Context(), r.URL.Query().Get("lang"))
	writeTax(w, rows, err)
}

func (h *Handler) searchCategories(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.SearchCategories(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("lang"), platform.QueryInt(r, "limit", 20))
	writeTax(w, rows, err)
}

func (h *Handler) categoriesByIDs(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.CategoriesByIDs(r.Context(), ParseIDs(r.URL.Query().Get("ids")), r.URL.Query().Get("lang"))
	writeTax(w, rows, err)
}

func (h *Handler) adminCategories(w http.ResponseWriter, r *http.Request) {
	page, err := h.store.AdminCategories(r.Context(), r.URL.Query().Get("lang"), platform.QueryInt(r, "page", 0), platform.QueryInt(r, "size", 10), r.URL.Query().Get("q"))
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, page)
}

func (h *Handler) createCategory(w http.ResponseWriter, r *http.Request) {
	var body TaxonomyIn
	if err := platform.DecodeJSONLenient(r, &body); err != nil {
		platform.WriteError(w, bad("invalid JSON body"), "error")
		return
	}
	item, err := h.store.CreateCategory(r.Context(), body)
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) updateCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r)
	if !ok {
		return
	}
	var body TaxonomyIn
	if err := platform.DecodeJSONLenient(r, &body); err != nil {
		platform.WriteError(w, bad("invalid JSON body"), "error")
		return
	}
	item, err := h.store.UpdateCategory(r.Context(), id, body)
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) deleteCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r)
	if !ok {
		return
	}
	if err := h.store.DeleteCategory(r.Context(), id); err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteNoContent(w)
}

func (h *Handler) listTags(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.ListPopularTags(r.Context(), r.URL.Query().Get("lang"))
	writeTax(w, rows, err)
}

func (h *Handler) searchTags(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.SearchTags(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("lang"), platform.QueryInt(r, "limit", 20))
	writeTax(w, rows, err)
}

func (h *Handler) tagsByIDs(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.TagsByIDs(r.Context(), ParseIDs(r.URL.Query().Get("ids")), r.URL.Query().Get("lang"))
	writeTax(w, rows, err)
}

func (h *Handler) adminTags(w http.ResponseWriter, r *http.Request) {
	page, err := h.store.AdminTags(r.Context(), r.URL.Query().Get("lang"), platform.QueryInt(r, "page", 0), platform.QueryInt(r, "size", 10), r.URL.Query().Get("q"))
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, page)
}

func (h *Handler) createTag(w http.ResponseWriter, r *http.Request) {
	var body TaxonomyIn
	if err := platform.DecodeJSONLenient(r, &body); err != nil {
		platform.WriteError(w, bad("invalid JSON body"), "error")
		return
	}
	item, err := h.store.CreateTag(r.Context(), body)
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) updateTag(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r)
	if !ok {
		return
	}
	var body TaxonomyIn
	if err := platform.DecodeJSONLenient(r, &body); err != nil {
		platform.WriteError(w, bad("invalid JSON body"), "error")
		return
	}
	item, err := h.store.UpdateTag(r.Context(), id, body)
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) deleteTag(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r)
	if !ok {
		return
	}
	if err := h.store.DeleteTag(r.Context(), id); err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteNoContent(w)
}

func (h *Handler) listComments(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.ListComments(r.Context(), r.URL.Query().Get("postId"), r.URL.Query().Get("status"))
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, rows)
}

func (h *Handler) createComment(w http.ResponseWriter, r *http.Request) {
	var body CreateComment
	if err := platform.DecodeJSONLenient(r, &body); err != nil {
		platform.WriteError(w, bad("invalid JSON body"), "error")
		return
	}
	c, err := h.store.CreateComment(r.Context(), body, subject(r))
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusCreated, c)
}

func (h *Handler) updateComment(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Content      string  `json:"content"`
		LanguageCode *string `json:"languageCode"`
	}
	if err := platform.DecodeJSONLenient(r, &body); err != nil {
		platform.WriteError(w, bad("invalid JSON body"), "error")
		return
	}
	c, err := h.store.UpdateComment(r.Context(), chi.URLParam(r, "id"), body.Content, body.LanguageCode)
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) getTranslation(w http.ResponseWriter, r *http.Request) {
	t, err := h.store.GetCommentTranslation(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "lang"))
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, t)
}

func (h *Handler) putTranslation(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	_ = platform.DecodeJSONLenient(r, &body)
	t, err := h.store.UpsertCommentTranslation(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "lang"), body["content"])
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, t)
}

func (h *Handler) patchComment(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Status *string `json:"status"`
	}
	_ = platform.DecodeJSONLenient(r, &body)
	status := ""
	if body.Status != nil {
		status = *body.Status
	}
	c, err := h.store.PatchCommentStatus(r.Context(), chi.URLParam(r, "id"), status)
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) deleteComment(w http.ResponseWriter, r *http.Request) {
	if err := h.store.DeleteComment(r.Context(), chi.URLParam(r, "id")); err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteNoContent(w)
}

func (h *Handler) getSettings(w http.ResponseWriter, r *http.Request) {
	body, err := h.store.GetSettings(r.Context(), r.URL.Query().Get("lang"))
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, body)
}

func (h *Handler) putSettings(w http.ResponseWriter, r *http.Request) {
	var body SiteSettingsIn
	if err := platform.DecodeJSONLenient(r, &body); err != nil {
		platform.WriteError(w, bad("settings body is required"), "error")
		return
	}
	out, err := h.store.PutSettings(r.Context(), body, r.URL.Query().Get("lang"))
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) getHero(w http.ResponseWriter, r *http.Request) {
	body, err := h.store.GetHomeHero(r.Context())
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, body)
}

func (h *Handler) putHero(w http.ResponseWriter, r *http.Request) {
	var body HomeHeroIn
	if err := platform.DecodeJSONLenient(r, &body); err != nil {
		platform.WriteError(w, bad("homeHero body is required"), "error")
		return
	}
	out, err := h.store.PutHomeHero(r.Context(), body)
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) getSections(w http.ResponseWriter, r *http.Request) {
	body, err := h.store.GetHomeSections(r.Context())
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, body)
}

func (h *Handler) putSections(w http.ResponseWriter, r *http.Request) {
	var body []VisItemIn
	if err := platform.DecodeJSONLenient(r, &body); err != nil {
		platform.WriteError(w, bad("homeSections body is required"), "error")
		return
	}
	out, err := h.store.PutHomeSections(r.Context(), body)
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) getMenu(w http.ResponseWriter, r *http.Request) {
	body, err := h.store.GetMainMenu(r.Context())
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, body)
}

func (h *Handler) putMenu(w http.ResponseWriter, r *http.Request) {
	var body []VisItemIn
	if err := platform.DecodeJSONLenient(r, &body); err != nil {
		platform.WriteError(w, bad("mainMenu body is required"), "error")
		return
	}
	out, err := h.store.PutMainMenu(r.Context(), body)
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) listUI(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.ListUIStrings(r.Context(), r.URL.Query().Get("lang"), r.URL.Query().Get("component"))
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, rows)
}

func (h *Handler) upsertUI(w http.ResponseWriter, r *http.Request) {
	var body struct {
		LanguageCode *string `json:"languageCode"`
		StringValue  *string `json:"stringValue"`
	}
	if err := platform.DecodeJSONLenient(r, &body); err != nil {
		platform.WriteError(w, bad("invalid JSON body"), "error")
		return
	}
	lang, val := "", ""
	if body.LanguageCode != nil {
		lang = *body.LanguageCode
	}
	if body.StringValue == nil {
		platform.WriteError(w, bad("stringValue is required"), "error")
		return
	}
	val = *body.StringValue
	item, err := h.store.UpsertUIString(r.Context(), chi.URLParam(r, "stringKey"), lang, val)
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) patchUI(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Items []struct {
			StringKey    string `json:"stringKey"`
			LanguageCode string `json:"languageCode"`
			StringValue  string `json:"stringValue"`
		} `json:"items"`
	}
	if err := platform.DecodeJSONLenient(r, &body); err != nil || body.Items == nil {
		platform.WriteError(w, bad("items are required"), "error")
		return
	}
	out := []UIString{}
	for _, item := range body.Items {
		row, err := h.store.UpsertUIString(r.Context(), item.StringKey, item.LanguageCode, item.StringValue)
		if err != nil {
			platform.WriteError(w, err, "error")
			return
		}
		out = append(out, row)
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

func writeTax(w http.ResponseWriter, rows []Taxonomy, err error) {
	if err != nil {
		platform.WriteError(w, err, "error")
		return
	}
	platform.WriteJSON(w, http.StatusOK, rows)
}

func pathInt(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		platform.WriteError(w, missing("not found"), "error")
		return 0, false
	}
	return id, true
}

type ctxKey int

const subjectKey ctxKey = 1

func subject(r *http.Request) string {
	v, _ := r.Context().Value(subjectKey).(string)
	return v
}

func (h *Handler) public(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			h.auth(next)(w, r)
			return
		}
		next(w, r)
	}
}

func (h *Handler) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := platform.BearerToken(r)
		if raw == "" {
			platform.WriteError(w, platform.Unauthorized("Unauthorized"), "error")
			return
		}
		claims, err := platform.ParseHS256(h.secret, h.issuer, raw)
		if err != nil || claims.Subject == "" {
			platform.WriteError(w, platform.Unauthorized("Unauthorized"), "error")
			return
		}
		ctx := context.WithValue(r.Context(), subjectKey, claims.Subject)
		next(w, r.WithContext(ctx))
	}
}

func init() {}
