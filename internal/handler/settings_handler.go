package handler

import (
	"net/http"

	"react-go-cms-content-service/internal/domain/apperror"
	"react-go-cms-content-service/internal/domain/entity"
	"react-go-cms-content-service/internal/infrastructure/httpx"
)

// SiteSettingsResponse is the public site settings document.
type SiteSettingsResponse struct {
	SiteName        string            `json:"siteName"`
	SiteIconURL     string            `json:"siteIconUrl"`
	SiteDescription string            `json:"siteDescription"`
	PostsPerPage    int               `json:"postsPerPage"`
	HomeHero        HomeHeroResponse  `json:"homeHero"`
	HomeSections    []VisItemResponse `json:"homeSections"`
	MainMenu        []VisItemResponse `json:"mainMenu"`
}

// HomeHeroResponse is the home hero block.
type HomeHeroResponse struct {
	TitleVisible    bool              `json:"titleVisible"`
	SubtitleVisible bool              `json:"subtitleVisible"`
	Ctas            []HeroCtaResponse `json:"ctas"`
}

// HeroCtaResponse is one home-hero call to action.
type HeroCtaResponse struct {
	ID              string            `json:"id"`
	Visible         bool              `json:"visible"`
	Href            string            `json:"href"`
	TextColor       string            `json:"textColor"`
	BackgroundColor string            `json:"backgroundColor"`
	Labels          map[string]string `json:"labels"`
}

// VisItemResponse is a home section or main-menu entry.
type VisItemResponse struct {
	ID        string `json:"id"`
	Visible   bool   `json:"visible"`
	ItemLimit *int   `json:"itemLimit"`
}

// SiteSettingsRequest is the settings write body.
type SiteSettingsRequest struct {
	SiteName        string            `json:"siteName"`
	SiteIconURL     *string           `json:"siteIconUrl"`
	SiteDescription *string           `json:"siteDescription"`
	PostsPerPage    int               `json:"postsPerPage"`
	HomeHero        *HomeHeroRequest  `json:"homeHero"`
	HomeSections    *[]VisItemRequest `json:"homeSections"`
	MainMenu        *[]VisItemRequest `json:"mainMenu"`
}

// HomeHeroRequest is a home-hero write body.
type HomeHeroRequest struct {
	TitleVisible    *bool            `json:"titleVisible"`
	SubtitleVisible *bool            `json:"subtitleVisible"`
	Ctas            []HeroCtaRequest `json:"ctas"`
}

// HeroCtaRequest is a hero call-to-action write body.
type HeroCtaRequest struct {
	ID              string            `json:"id"`
	Visible         *bool             `json:"visible"`
	Href            *string           `json:"href"`
	TextColor       *string           `json:"textColor"`
	BackgroundColor *string           `json:"backgroundColor"`
	Labels          map[string]string `json:"labels"`
}

// VisItemRequest is a home-section or menu write body.
type VisItemRequest struct {
	ID        string `json:"id"`
	Visible   *bool  `json:"visible"`
	ItemLimit *int   `json:"itemLimit"`
}

func (h *Handler) getSettings(w http.ResponseWriter, r *http.Request) {
	var body entity.SiteSettings
	var err error
	body, err = h.settings.Get(r.Context(), r.URL.Query().Get("lang"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toSiteSettingsResponse(body))
}

func (h *Handler) putSettings(w http.ResponseWriter, r *http.Request) {
	var body SiteSettingsRequest
	var err error
	err = httpx.DecodeJSONLenient(r, &body)
	if err != nil {
		writeServiceError(w, apperror.Invalid("settings body is required"))
		return
	}
	var out entity.SiteSettings
	out, err = h.settings.Put(r.Context(), toSiteSettingsInput(body), r.URL.Query().Get("lang"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toSiteSettingsResponse(out))
}

func (h *Handler) getHero(w http.ResponseWriter, r *http.Request) {
	var body entity.HomeHero
	var err error
	body, err = h.settings.Hero(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toHomeHeroResponse(body))
}

func (h *Handler) putHero(w http.ResponseWriter, r *http.Request) {
	var body HomeHeroRequest
	var err error
	err = httpx.DecodeJSONLenient(r, &body)
	if err != nil {
		writeServiceError(w, apperror.Invalid("homeHero body is required"))
		return
	}
	var out entity.HomeHero
	out, err = h.settings.PutHero(r.Context(), toHomeHeroInput(body))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toHomeHeroResponse(out))
}

func (h *Handler) getSections(w http.ResponseWriter, r *http.Request) {
	var body []entity.VisItem
	var err error
	body, err = h.settings.HomeSections(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toVisItemResponses(body))
}

func (h *Handler) putSections(w http.ResponseWriter, r *http.Request) {
	var body []VisItemRequest
	var err error
	err = httpx.DecodeJSONLenient(r, &body)
	if err != nil {
		writeServiceError(w, apperror.Invalid("homeSections body is required"))
		return
	}
	var out []entity.VisItem
	out, err = h.settings.PutHomeSections(r.Context(), toVisItemInputs(body))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toVisItemResponses(out))
}

func (h *Handler) getMenu(w http.ResponseWriter, r *http.Request) {
	var body []entity.VisItem
	var err error
	body, err = h.settings.MainMenu(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toVisItemResponses(body))
}

func (h *Handler) putMenu(w http.ResponseWriter, r *http.Request) {
	var body []VisItemRequest
	var err error
	err = httpx.DecodeJSONLenient(r, &body)
	if err != nil {
		writeServiceError(w, apperror.Invalid("mainMenu body is required"))
		return
	}
	var out []entity.VisItem
	out, err = h.settings.PutMainMenu(r.Context(), toVisItemInputs(body))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toVisItemResponses(out))
}

func toSiteSettingsResponse(item entity.SiteSettings) SiteSettingsResponse {
	return SiteSettingsResponse{
		SiteName:        item.SiteName,
		SiteIconURL:     item.SiteIconURL,
		SiteDescription: item.SiteDescription,
		PostsPerPage:    item.PostsPerPage,
		HomeHero:        toHomeHeroResponse(item.HomeHero),
		HomeSections:    toVisItemResponses(item.HomeSections),
		MainMenu:        toVisItemResponses(item.MainMenu),
	}
}

func toHomeHeroResponse(item entity.HomeHero) HomeHeroResponse {
	ctas := make([]HeroCtaResponse, 0, len(item.Ctas))
	for _, cta := range item.Ctas {
		labels := cta.Labels
		if labels == nil {
			labels = map[string]string{}
		}
		ctas = append(ctas, HeroCtaResponse{
			ID:              cta.ID,
			Visible:         cta.Visible,
			Href:            cta.Href,
			TextColor:       cta.TextColor,
			BackgroundColor: cta.BackgroundColor,
			Labels:          labels,
		})
	}
	return HomeHeroResponse{TitleVisible: item.TitleVisible, SubtitleVisible: item.SubtitleVisible, Ctas: ctas}
}

func toVisItemResponses(items []entity.VisItem) []VisItemResponse {
	if items == nil {
		return []VisItemResponse{}
	}
	out := make([]VisItemResponse, 0, len(items))
	for _, item := range items {
		out = append(out, VisItemResponse{ID: item.ID, Visible: item.Visible, ItemLimit: item.ItemLimit})
	}
	return out
}

func toSiteSettingsInput(body SiteSettingsRequest) entity.SiteSettingsInput {
	input := entity.SiteSettingsInput{
		SiteName:        body.SiteName,
		SiteIconURL:     body.SiteIconURL,
		SiteDescription: body.SiteDescription,
		PostsPerPage:    body.PostsPerPage,
	}
	if body.HomeHero != nil {
		hero := toHomeHeroInput(*body.HomeHero)
		input.HomeHero = &hero
	}
	if body.HomeSections != nil {
		sections := toVisItemInputs(*body.HomeSections)
		input.HomeSections = &sections
	}
	if body.MainMenu != nil {
		menu := toVisItemInputs(*body.MainMenu)
		input.MainMenu = &menu
	}
	return input
}

func toHomeHeroInput(body HomeHeroRequest) entity.HomeHeroInput {
	ctas := make([]entity.HeroCtaInput, 0, len(body.Ctas))
	for _, cta := range body.Ctas {
		ctas = append(ctas, entity.HeroCtaInput{
			ID:              cta.ID,
			Visible:         cta.Visible,
			Href:            cta.Href,
			TextColor:       cta.TextColor,
			BackgroundColor: cta.BackgroundColor,
			Labels:          cta.Labels,
		})
	}
	return entity.HomeHeroInput{TitleVisible: body.TitleVisible, SubtitleVisible: body.SubtitleVisible, Ctas: ctas}
}

func toVisItemInputs(items []VisItemRequest) []entity.VisItemInput {
	out := make([]entity.VisItemInput, 0, len(items))
	for _, item := range items {
		out = append(out, entity.VisItemInput{ID: item.ID, Visible: item.Visible, ItemLimit: item.ItemLimit})
	}
	return out
}
