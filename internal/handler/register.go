package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"react-go-cms-content-service/internal/application/service/category"
	"react-go-cms-content-service/internal/application/service/comment"
	"react-go-cms-content-service/internal/application/service/contenttype"
	"react-go-cms-content-service/internal/application/service/dashboard"
	"react-go-cms-content-service/internal/application/service/post"
	"react-go-cms-content-service/internal/application/service/settings"
	"react-go-cms-content-service/internal/application/service/tag"
	"react-go-cms-content-service/internal/application/service/uistring"
	"react-go-cms-content-service/internal/domain/apperror"
	"react-go-cms-content-service/internal/infrastructure/httpx"
)

const (
	healthBody       = "content-service-ok"
	errorField       = "error"
	invalidJSON      = "invalid JSON body"
	defaultPage      = 0
	defaultPageSize  = 20
	defaultAdminSize = 10
	defaultLimit     = 20
	defaultRecent    = 6
)

// Handler exposes the content service HTTP endpoints.
type Handler struct {
	posts        *post.Service
	categories   *category.Service
	tags         *tag.Service
	comments     *comment.Service
	contentTypes *contenttype.Service
	dashboard    *dashboard.Service
	settings     *settings.Service
	uiStrings    *uistring.Service
	secret       string
	issuer       string
}

// New builds the HTTP handler.
func New(posts *post.Service, categories *category.Service, tags *tag.Service, comments *comment.Service, contentTypes *contenttype.Service, dashboardService *dashboard.Service, settingsService *settings.Service, uiStrings *uistring.Service, secret, issuer string) *Handler {
	return &Handler{
		posts:        posts,
		categories:   categories,
		tags:         tags,
		comments:     comments,
		contentTypes: contentTypes,
		dashboard:    dashboardService,
		settings:     settingsService,
		uiStrings:    uiStrings,
		secret:       secret,
		issuer:       issuer,
	}
}

// Register wires the content service routes.
func (h *Handler) Register(mux chi.Router) {
	mux.Get("/api/content/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(healthBody))
	})

	mux.Get("/api/admin/dashboard", h.auth(h.dashboardPage))
	mux.Get("/api/content-types", h.contentTypesList)

	mux.Get("/api/posts/by-slug/{slug}", h.postBySlug)
	mux.Get("/api/posts/{id}/metadata", h.getMetadata)
	mux.Put("/api/posts/{id}/metadata", h.auth(h.putMetadata))
	mux.Patch("/api/posts/{id}/status", h.auth(h.patchPostStatus))
	mux.Post("/api/posts/{id}/view", h.auth(h.viewPost))
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

	mux.Get("/api/comments/{id}/translations/{lang}", h.getCommentTranslation)
	mux.Put("/api/comments/{id}/translations/{lang}", h.auth(h.putCommentTranslation))
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

func writeServiceError(w http.ResponseWriter, err error) {
	var domainErr *apperror.Error
	if errors.As(err, &domainErr) {
		status := http.StatusBadRequest
		if errors.Is(err, apperror.ErrNotFound) {
			status = http.StatusNotFound
		}
		httpx.WriteJSON(w, status, map[string]string{errorField: domainErr.Message})
		return
	}
	httpx.WriteError(w, err, errorField)
}

func pathInt(w http.ResponseWriter, r *http.Request) (int, bool) {
	var id int
	var err error
	id, err = strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, apperror.NotFound("not found"))
		return 0, false
	}
	return id, true
}

func parseIDs(csv string) []int {
	if strings.TrimSpace(csv) == "" {
		return []int{}
	}
	out := []int{}
	for _, part := range strings.Split(csv, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		value := 0
		ok := true
		for _, char := range part {
			if char < '0' || char > '9' {
				ok = false
				break
			}
			value = value*10 + int(char-'0')
		}
		if ok {
			out = append(out, value)
		}
	}
	return out
}
