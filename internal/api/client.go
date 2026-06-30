// Package api contains the thin HTTP client used by CLI commands.
// Generated or hand-mapped endpoint clients should build on this package,
// not duplicate auth headers or URL construction.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

type Options struct {
	APIURL string
	APIKey string
}

type MeResponse struct {
	User struct {
		ID    string  `json:"id"`
		Email string  `json:"email"`
		Name  *string `json:"name"`
	} `json:"user"`
	AccessibleSpaces []SpaceSummary `json:"accessibleSpaces"`
	APIKey           struct {
		ID     string `json:"id"`
		Prefix string `json:"prefix"`
	} `json:"apiKey"`
}

type SpaceSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type Domain struct {
	ID       string  `json:"id"`
	Hostname string  `json:"hostname"`
	Type     string  `json:"type"`
	Tier     *string `json:"tier"`
}

type ListDomainsResponse struct {
	Domains []Domain `json:"domains"`
}

type Collection struct {
	ID          string  `json:"id"`
	SpaceID     string  `json:"spaceId"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Type        string  `json:"type"`
	LinkCount   int     `json:"linkCount"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   *string `json:"updatedAt"`
}

type ListCollectionsResponse struct {
	Collections []Collection `json:"collections"`
}

type CreateCollectionInput struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type CreateCollectionResponse struct {
	Collection Collection `json:"collection"`
}

type Link struct {
	ID            string  `json:"id"`
	SpaceID       string  `json:"spaceId"`
	ShortDomainID string  `json:"shortDomainId"`
	Hostname      string  `json:"hostname"`
	Path          string  `json:"path"`
	ShortURL      string  `json:"shortUrl,omitempty"`
	TargetURL     string  `json:"targetUrl"`
	Title         *string `json:"title"`
	Description   *string `json:"description"`
	IsActive      bool    `json:"isActive"`
	CreatedAt     string  `json:"createdAt"`
}

type CreateLinkInput struct {
	TargetURL   string `json:"targetUrl"`
	Domain      string `json:"domain,omitempty"`
	Path        string `json:"path,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Collection  string `json:"collection,omitempty"`
}

type CreateLinkResponse struct {
	Link   Link   `json:"link"`
	Source string `json:"source"`
	// Advisory reachability of the target URL: non-nil only when the create
	// was made with verification on (?verify=true). true = resolved to a live
	// page, false = unreachable, nil = not checked. Never gates creation.
	TargetReachable *bool `json:"targetReachable"`
}

type ListLinksResponse struct {
	Links      []Link  `json:"links"`
	NextCursor *string `json:"nextCursor"`
}

type ListLinksOptions struct {
	Limit  int
	Cursor string
	Sort   string
	Status string
}

type ErrorResponse struct {
	Error struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details,omitempty"`
	} `json:"error"`
}

func New(options Options) *Client {
	return &Client{
		baseURL: strings.TrimRight(options.APIURL, "/"),
		apiKey:  options.APIKey,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (c *Client) GetMe(ctx context.Context) (MeResponse, error) {
	var response MeResponse
	err := c.DoJSON(ctx, http.MethodGet, "/me", nil, &response)
	return response, err
}

func (c *Client) ListDomains(ctx context.Context, spaceID string) (ListDomainsResponse, error) {
	var response ListDomainsResponse
	err := c.DoJSON(ctx, http.MethodGet, "/spaces/"+url.PathEscape(spaceID)+"/domains", nil, &response)
	return response, err
}

func (c *Client) ListCollections(ctx context.Context, spaceID string) (ListCollectionsResponse, error) {
	var response ListCollectionsResponse
	err := c.DoJSON(ctx, http.MethodGet, "/spaces/"+url.PathEscape(spaceID)+"/collections", nil, &response)
	return response, err
}

func (c *Client) CreateCollection(ctx context.Context, spaceID string, input CreateCollectionInput) (CreateCollectionResponse, error) {
	var response CreateCollectionResponse
	err := c.DoJSON(ctx, http.MethodPost, "/spaces/"+url.PathEscape(spaceID)+"/collections", input, &response)
	return response, err
}

func (c *Client) ListLinks(ctx context.Context, spaceID string, options ListLinksOptions) (ListLinksResponse, error) {
	var response ListLinksResponse
	err := c.DoJSON(ctx, http.MethodGet, "/spaces/"+url.PathEscape(spaceID)+"/links"+queryString(options), nil, &response)
	return response, err
}

func (c *Client) CreateLink(ctx context.Context, spaceID string, input CreateLinkInput, verify bool) (CreateLinkResponse, error) {
	var response CreateLinkResponse
	path := "/spaces/" + url.PathEscape(spaceID) + "/links"
	if verify {
		// Ask the API to probe the target and report targetReachable. Adds a
		// round-trip server-side; the link is created either way.
		path += "?verify=true"
	}
	err := c.DoJSON(ctx, http.MethodPost, path, input, &response)
	return response, err
}

func (c *Client) ListCollectionLinks(ctx context.Context, spaceID string, collectionID string, options ListLinksOptions) (ListLinksResponse, error) {
	var response ListLinksResponse
	path := "/spaces/" + url.PathEscape(spaceID) + "/collections/" + url.PathEscape(collectionID) + "/links"
	err := c.DoJSON(ctx, http.MethodGet, path+queryString(options), nil, &response)
	return response, err
}

func (c *Client) DoJSON(ctx context.Context, method string, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var apiErr ErrorResponse
		if err := json.Unmarshal(data, &apiErr); err == nil && apiErr.Error.Message != "" {
			return fmt.Errorf("%s: %s", apiErr.Error.Code, apiErr.Error.Message)
		}
		return fmt.Errorf("HTTP %d from %s %s: %s", res.StatusCode, method, path, strings.TrimSpace(string(data)))
	}
	if out == nil || res.StatusCode == http.StatusNoContent {
		return nil
	}
	return json.Unmarshal(data, out)
}

func queryString(options ListLinksOptions) string {
	values := url.Values{}
	if options.Limit > 0 {
		values.Set("limit", fmt.Sprintf("%d", options.Limit))
	}
	if options.Cursor != "" {
		values.Set("cursor", options.Cursor)
	}
	if options.Sort != "" {
		values.Set("sort", options.Sort)
	}
	if options.Status != "" {
		values.Set("status", options.Status)
	}
	if len(values) == 0 {
		return ""
	}
	return "?" + values.Encode()
}
