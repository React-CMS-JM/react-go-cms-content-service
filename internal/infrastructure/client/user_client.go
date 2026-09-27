// Package client calls the auth service for dashboard author names.
package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"react-go-cms-content-service/internal/application/service/dashboard"
)

const userClientTimeout = 5 * time.Second

type userSummary struct {
	ID        string  `json:"id"`
	FirstName *string `json:"firstName"`
	LastName  *string `json:"lastName"`
}

// UserClient loads display names from GET {AUTH_SERVICE_URL}/api/users/by-ids.
type UserClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewUserClient builds the auth-service user client.
func NewUserClient(authServiceURL string) *UserClient {
	return &UserClient{
		baseURL:    strings.TrimRight(authServiceURL, "/"),
		httpClient: &http.Client{Timeout: userClientTimeout},
	}
}

var _ dashboard.UserClient = (*UserClient)(nil)

// AuthorNames returns trimmed "first last" names. Failures yield an empty map.
func (c *UserClient) AuthorNames(ctx context.Context, ids []string, authorization string) map[string]string {
	out := map[string]string{}
	if len(ids) == 0 || strings.TrimSpace(authorization) == "" {
		return out
	}
	seen := map[string]struct{}{}
	uniq := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	var req *http.Request
	var err error
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/users/by-ids?ids="+url.QueryEscape(strings.Join(uniq, ",")), nil)
	if err != nil {
		return out
	}
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Accept", "application/json")
	var res *http.Response
	res, err = c.httpClient.Do(req)
	if err != nil {
		return out
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return out
	}
	var body []byte
	body, err = io.ReadAll(res.Body)
	if err != nil {
		return out
	}
	var rows []userSummary
	if json.Unmarshal(body, &rows) != nil {
		return out
	}
	for _, user := range rows {
		first, last := "", ""
		if user.FirstName != nil {
			first = strings.TrimSpace(*user.FirstName)
		}
		if user.LastName != nil {
			last = strings.TrimSpace(*user.LastName)
		}
		out[user.ID] = strings.TrimSpace(first + " " + last)
	}
	return out
}
