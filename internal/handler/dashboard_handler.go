package handler

import (
	"fmt"
	"net/http"

	"react-go-cms-content-service/internal/domain/entity"
	"react-go-cms-content-service/internal/infrastructure/httpx"
)

// TypeCountsResponse is the dashboard content-type tally.
type TypeCountsResponse struct {
	Post    int64
	Page    int64
	Service int64
	Product int64
}

// MarshalJSON keeps the dashboard count object field names.
func (c TypeCountsResponse) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, `{"post":%d,"page":%d,"service":%d,"product":%d}`, c.Post, c.Page, c.Service, c.Product), nil
}

// RecentResponse is one dashboard recent post.
type RecentResponse struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	AuthorID        string    `json:"authorId"`
	AuthorName      string    `json:"authorName"`
	Status          string    `json:"status"`
	ContentTypeSlug string    `json:"contentTypeSlug"`
	UpdatedAt       LocalJSON `json:"updatedAt"`
}

// DashboardResponse is the admin dashboard payload.
type DashboardResponse struct {
	Counts          TypeCountsResponse `json:"counts"`
	PendingComments int64              `json:"pendingComments"`
	Recent          []RecentResponse   `json:"recent"`
}

func (h *Handler) dashboardPage(w http.ResponseWriter, r *http.Request) {
	var item entity.Dashboard
	var err error
	item, err = h.dashboard.Load(r.Context(), r.URL.Query().Get("lang"), httpx.QueryInt(r, "recentLimit", defaultRecent), r.Header.Get("Authorization"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toDashboardResponse(item))
}

func toDashboardResponse(item entity.Dashboard) DashboardResponse {
	recent := make([]RecentResponse, 0, len(item.Recent))
	for _, row := range item.Recent {
		recent = append(recent, RecentResponse{
			ID:              row.ID,
			Title:           row.Title,
			AuthorID:        row.AuthorID,
			AuthorName:      row.AuthorName,
			Status:          row.Status,
			ContentTypeSlug: row.ContentTypeSlug,
			UpdatedAt:       toLocal(row.UpdatedAt),
		})
	}
	return DashboardResponse{
		Counts: TypeCountsResponse{
			Post:    item.Counts.Post,
			Page:    item.Counts.Page,
			Service: item.Counts.Service,
			Product: item.Counts.Product,
		},
		PendingComments: item.PendingComments,
		Recent:          recent,
	}
}
