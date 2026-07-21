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

// CreateCollectionInput creates a manual collection ({name}) or a smart one
// ({type:"smart", name, filter}). Leaving Type empty and Filter nil creates a
// manual collection; setting Type "smart" with a Filter creates a live,
// rule-based collection whose membership equals a query of the same filter.
type CreateCollectionInput struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Type        string      `json:"type,omitempty"`
	Filter      *LinkFilter `json:"filter,omitempty"`
}

type CreateCollectionResponse struct {
	Collection Collection `json:"collection"`
	// Set for smart collections: a plain-language echo of the compiled
	// membership rule (e.g. "Active links not clicked in the last 30 days").
	RulesSummary *string `json:"rulesSummary,omitempty"`
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
	// Present on list rows when the request used a click sort or
	// include=clicks; nil otherwise.
	Clicks        *int    `json:"clicks,omitempty"`
	LastClickedAt *string `json:"lastClickedAt,omitempty"`
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
	Link Link `json:"link"`
	// Source is set by the single-create endpoint only; rows synthesized
	// from bulk results leave it empty and it is omitted from output.
	Source string `json:"source,omitempty"`
	// Advisory reachability of the target URL: non-nil only when the create
	// was made with verification on (?verify=true). true = resolved to a live
	// page, false = unreachable, nil = not checked. Never gates creation.
	TargetReachable *bool `json:"targetReachable"`
}

// ClickThreshold is a click-count comparison used by the click filter
// dimensions: {op: greaterThan|lessThan, value: N}.
type ClickThreshold struct {
	Op    string `json:"op"`
	Value int    `json:"value"`
}

// LinkFilter is the one filter vocabulary, mirroring Core's LinkFilter. Every
// field is optional; dimensions AND-combine, arrays OR within a dimension, and
// Negate inverts named dimensions. The two host dimensions take hostnames, not
// ids. Contract: docs/UNIVERSAL-QUERY-FILTER-SYSTEM.md.
type LinkFilter struct {
	Query         string          `json:"query,omitempty"`
	Status        string          `json:"status,omitempty"`
	Created       string          `json:"created,omitempty"`
	Edited        string          `json:"edited,omitempty"`
	Clicked       string          `json:"clicked,omitempty"`
	Schedule      string          `json:"schedule,omitempty"`
	CreatedVia    string          `json:"createdVia,omitempty"`
	Attribution   string          `json:"attribution,omitempty"`
	HasCollection *bool           `json:"hasCollection,omitempty"`
	ShortDomain   []string        `json:"shortDomain,omitempty"`
	TargetHost    []string        `json:"targetHost,omitempty"`
	Clicks        *ClickThreshold `json:"clicks,omitempty"`
	UniqueClicks  *ClickThreshold `json:"uniqueClicks,omitempty"`
	Negate        []string        `json:"negate,omitempty"`
}

// QueryLinksInput is the POST /links/query body: a LinkFilter plus paging. The
// embedded LinkFilter flattens into the JSON body.
type QueryLinksInput struct {
	LinkFilter
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// QueryLinksResponse is a query result: a page of matching links plus the true
// uncapped match count.
type QueryLinksResponse struct {
	Links []Link `json:"links"`
	Total int    `json:"total"`
}

// LookupLinkResponse resolves a short URL/code to the one link it addresses.
type LookupLinkResponse struct {
	Link Link `json:"link"`
}

// AnalyticsQueryInput — the analytics query body. Two halves: object SCOPE
// (which links to count clicks for — the same vocabulary as LinkFilter's object
// dims) and click MEASUREMENT (which clicks), plus groupBy/measure/range.
// Mirrors Core's QueryAnalyticsInput. Contract: docs/ANALYTICS-QUERY-SYSTEM.md.
type AnalyticsQueryInput struct {
	// Object scope — which links.
	Query         string   `json:"query,omitempty"`
	Status        string   `json:"status,omitempty"`
	Created       string   `json:"created,omitempty"`
	Edited        string   `json:"edited,omitempty"`
	CreatedVia    string   `json:"createdVia,omitempty"`
	HasCollection *bool    `json:"hasCollection,omitempty"`
	Schedule      string   `json:"schedule,omitempty"`
	Attribution   string   `json:"attribution,omitempty"`
	TargetHost    []string `json:"targetHost,omitempty"`
	CollectionID  string   `json:"collectionId,omitempty"`
	LinkID        string   `json:"linkId,omitempty"`
	Negate        []string `json:"negate,omitempty"`
	// Click measurement — which clicks.
	Country        []string `json:"country,omitempty"`
	Continents     []string `json:"continents,omitempty"`
	Region         []string `json:"region,omitempty"`
	City           []string `json:"city,omitempty"`
	Browser        []string `json:"browser,omitempty"`
	OS             []string `json:"os,omitempty"`
	DeviceType     []string `json:"deviceType,omitempty"`
	ReferrerDomain []string `json:"referrerDomain,omitempty"`
	ShortDomain    []string `json:"shortDomain,omitempty"`
	FromQr         *bool    `json:"fromQr,omitempty"`
	IsBot          *bool    `json:"isBot,omitempty"`
	BotType        []string `json:"botType,omitempty"`
	BotName        []string `json:"botName,omitempty"`
	// Aggregation.
	GroupBy string `json:"groupBy,omitempty"`
	Measure string `json:"measure,omitempty"`
	Range   string `json:"range,omitempty"`
	From    string `json:"from,omitempty"`
	To      string `json:"to,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

// AnalyticsRow is one breakdown row (or the single total when key is null).
type AnalyticsRow struct {
	Key          *string `json:"key"`
	Clicks       int     `json:"clicks"`
	UniqueClicks int     `json:"uniqueClicks"`
}

// AnalyticsQueryResponse — aggregate rows plus the configured/tooLarge flags.
type AnalyticsQueryResponse struct {
	Configured bool           `json:"configured"`
	TooLarge   bool           `json:"tooLarge"`
	Range      string         `json:"range"`
	GroupBy    *string        `json:"groupBy"`
	Measure    *string        `json:"measure"`
	Message    string         `json:"message,omitempty"`
	Rows       []AnalyticsRow `json:"rows"`
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
	// IncludeClicks asks the API for clicks/lastClickedAt on every row
	// (include=clicks). Click sorts include them regardless.
	IncludeClicks bool
}

type GetLinkResponse struct {
	Link Link `json:"link"`
}

// UpdateLinkInput is the PATCH body. It is a plain map so commands can send
// exactly the fields the user asked to change — including explicit nulls
// (e.g. clearing a title), which typed omitempty structs cannot express.
type UpdateLinkInput map[string]any

type UpdateLinkResponse struct {
	Link        Link `json:"link"`
	PathChanged bool `json:"pathChanged"`
}

type BulkRowError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type BulkCreateLinkItem struct {
	TargetURL string `json:"targetUrl"`
	Domain    string `json:"domain,omitempty"`
	Path      string `json:"path,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Title     string `json:"title,omitempty"`
}

type BulkCreateLinksInput struct {
	Collection string               `json:"collection,omitempty"`
	Items      []BulkCreateLinkItem `json:"items"`
}

type BulkCreateRowResult struct {
	Index   int           `json:"index"`
	Success bool          `json:"success"`
	Link    *Link         `json:"link,omitempty"`
	Error   *BulkRowError `json:"error,omitempty"`
}

type BulkCreateLinksResponse struct {
	Results []BulkCreateRowResult `json:"results"`
}

type BulkDeleteRowResult struct {
	LinkID  string        `json:"linkId"`
	Success bool          `json:"success"`
	Error   *BulkRowError `json:"error,omitempty"`
}

type BulkDeleteLinksResponse struct {
	Results []BulkDeleteRowResult `json:"results"`
}

type UpdateCollectionInput struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

type CollectionResponse struct {
	Collection Collection `json:"collection"`
}

type DeleteCollectionResponse struct {
	DeletedCollectionID string `json:"deletedCollectionId"`
}

type AddCollectionLinksResponse struct {
	Added         int `json:"added"`
	AlreadyMember int `json:"alreadyMember"`
}

type RemoveCollectionLinksResponse struct {
	Removed int `json:"removed"`
}

type HealthResponse struct {
	OK  bool   `json:"ok"`
	API string `json:"api"`
}

// QrImageUrls are a design's stable, key-free public image URLs — the same
// files an <img> tag or third party can embed. Saving the design rewrites them
// in place, so each URL always serves the latest look.
type QrImageUrls struct {
	PNG string `json:"png"`
	SVG string `json:"svg"`
}

// QrVariant is a named QR design on a link. Style/signals are the studio's
// authoring vocabulary; the CLI treats them as opaque JSON (it reads variants,
// it doesn't author them).
type QrVariant struct {
	ID        string          `json:"id"`
	LinkID    string          `json:"linkId"`
	Name      string          `json:"name"`
	Style     json.RawMessage `json:"style"`
	Signals   json.RawMessage `json:"signals"`
	ImageUrls *QrImageUrls    `json:"imageUrls"`
	CreatedAt string          `json:"createdAt"`
	UpdatedAt *string         `json:"updatedAt"`
}

type ListQrVariantsResponse struct {
	QrVariants        []QrVariant     `json:"qrVariants"`
	SpaceDefaultStyle json.RawMessage `json:"spaceDefaultStyle"`
}

type QrExportResponse struct {
	Export struct {
		ImageUrls   QrImageUrls `json:"imageUrls"`
		VariantID   *string     `json:"variantId"`
		VariantName *string     `json:"variantName"`
	} `json:"export"`
}

type ErrorResponse struct {
	Error struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details,omitempty"`
	} `json:"error"`
}

// APIError is a structured, non-2xx response from Core. Commands return it up
// to the root, which renders it as JSON under --json (preserving the machine
// -readable code) or as a `zeb: CODE: message` line otherwise. Errors from the
// API carry a Code; transport/HTTP failures with no JSON body leave it empty.
type APIError struct {
	Status  int
	Code    string
	Message string
	Details map[string]any
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return e.Message
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

func (c *Client) QueryLinks(ctx context.Context, spaceID string, input QueryLinksInput) (QueryLinksResponse, error) {
	var response QueryLinksResponse
	err := c.DoJSON(ctx, http.MethodPost, "/spaces/"+url.PathEscape(spaceID)+"/links/query", input, &response)
	return response, err
}

// LookupLink resolves a short link to its record via GET /links/lookup. Pass a
// full short URL, or a code (key) with its domain; an unknown link surfaces as
// a 404 *APIError.
func (c *Client) LookupLink(ctx context.Context, spaceID string, shortURL string, domain string, key string) (LookupLinkResponse, error) {
	params := url.Values{}
	if shortURL != "" {
		params.Set("url", shortURL)
	}
	if domain != "" {
		params.Set("domain", domain)
	}
	if key != "" {
		params.Set("key", key)
	}
	path := "/spaces/" + url.PathEscape(spaceID) + "/links/lookup"
	if encoded := params.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var response LookupLinkResponse
	err := c.DoJSON(ctx, http.MethodGet, path, nil, &response)
	return response, err
}

func (c *Client) QueryAnalytics(ctx context.Context, spaceID string, input AnalyticsQueryInput) (AnalyticsQueryResponse, error) {
	var response AnalyticsQueryResponse
	err := c.DoJSON(ctx, http.MethodPost, "/spaces/"+url.PathEscape(spaceID)+"/analytics/query", input, &response)
	return response, err
}

func (c *Client) ListCollectionLinks(ctx context.Context, spaceID string, collectionID string, options ListLinksOptions) (ListLinksResponse, error) {
	var response ListLinksResponse
	path := "/spaces/" + url.PathEscape(spaceID) + "/collections/" + url.PathEscape(collectionID) + "/links"
	err := c.DoJSON(ctx, http.MethodGet, path+queryString(options), nil, &response)
	return response, err
}

func (c *Client) GetLink(ctx context.Context, spaceID string, linkID string) (GetLinkResponse, error) {
	var response GetLinkResponse
	err := c.DoJSON(ctx, http.MethodGet, c.linkPath(spaceID, linkID), nil, &response)
	return response, err
}

func (c *Client) UpdateLink(ctx context.Context, spaceID string, linkID string, input UpdateLinkInput) (UpdateLinkResponse, error) {
	var response UpdateLinkResponse
	err := c.DoJSON(ctx, http.MethodPatch, c.linkPath(spaceID, linkID), input, &response)
	return response, err
}

func (c *Client) BulkCreateLinks(ctx context.Context, spaceID string, input BulkCreateLinksInput) (BulkCreateLinksResponse, error) {
	var response BulkCreateLinksResponse
	err := c.DoJSON(ctx, http.MethodPost, "/spaces/"+url.PathEscape(spaceID)+"/links/bulk", input, &response)
	return response, err
}

func (c *Client) BulkDeleteLinks(ctx context.Context, spaceID string, linkIDs []string) (BulkDeleteLinksResponse, error) {
	var response BulkDeleteLinksResponse
	body := map[string][]string{"linkIds": linkIDs}
	err := c.DoJSON(ctx, http.MethodDelete, "/spaces/"+url.PathEscape(spaceID)+"/links/bulk", body, &response)
	return response, err
}

func (c *Client) GetCollection(ctx context.Context, spaceID string, collectionID string) (CollectionResponse, error) {
	var response CollectionResponse
	err := c.DoJSON(ctx, http.MethodGet, c.collectionPath(spaceID, collectionID), nil, &response)
	return response, err
}

func (c *Client) UpdateCollection(ctx context.Context, spaceID string, collectionID string, input UpdateCollectionInput) (CollectionResponse, error) {
	var response CollectionResponse
	err := c.DoJSON(ctx, http.MethodPatch, c.collectionPath(spaceID, collectionID), input, &response)
	return response, err
}

func (c *Client) DeleteCollection(ctx context.Context, spaceID string, collectionID string) (DeleteCollectionResponse, error) {
	var response DeleteCollectionResponse
	err := c.DoJSON(ctx, http.MethodDelete, c.collectionPath(spaceID, collectionID), nil, &response)
	return response, err
}

func (c *Client) ConvertCollectionToManual(ctx context.Context, spaceID string, collectionID string) (CollectionResponse, error) {
	var response CollectionResponse
	err := c.DoJSON(ctx, http.MethodPost, c.collectionPath(spaceID, collectionID)+"/convert-to-manual", nil, &response)
	return response, err
}

func (c *Client) AddLinksToCollection(ctx context.Context, spaceID string, collectionID string, linkIDs []string) (AddCollectionLinksResponse, error) {
	var response AddCollectionLinksResponse
	body := map[string][]string{"linkIds": linkIDs}
	err := c.DoJSON(ctx, http.MethodPost, c.collectionPath(spaceID, collectionID)+"/links", body, &response)
	return response, err
}

func (c *Client) RemoveLinksFromCollection(ctx context.Context, spaceID string, collectionID string, linkIDs []string) (RemoveCollectionLinksResponse, error) {
	var response RemoveCollectionLinksResponse
	body := map[string][]string{"linkIds": linkIDs}
	err := c.DoJSON(ctx, http.MethodDelete, c.collectionPath(spaceID, collectionID)+"/links", body, &response)
	return response, err
}

func (c *Client) Health(ctx context.Context) (HealthResponse, error) {
	var response HealthResponse
	err := c.DoJSON(ctx, http.MethodGet, "/health", nil, &response)
	return response, err
}

// ListQrVariants returns a link's named QR designs (the first is its effective
// default) plus the space default style.
func (c *Client) ListQrVariants(ctx context.Context, spaceID string, linkID string) (ListQrVariantsResponse, error) {
	var response ListQrVariantsResponse
	err := c.DoJSON(ctx, http.MethodGet, c.linkPath(spaceID, linkID)+"/qr-variants", nil, &response)
	return response, err
}

// QrImageOptions selects how a link's QR image renders. Zero values inherit the
// server defaults (PNG, default size, effective-default design).
type QrImageOptions struct {
	Format  string // "png" (default) or "svg"
	Size    int    // PNG edge length in px; ignored for SVG
	Variant string // render a named variant instead of the effective default
}

// GetQrImage renders a link's QR code and returns the raw bytes plus their
// Content-Type. The image encodes the link's canonical short URL.
func (c *Client) GetQrImage(ctx context.Context, spaceID string, linkID string, opts QrImageOptions) ([]byte, string, error) {
	values := url.Values{}
	if opts.Format != "" {
		values.Set("format", opts.Format)
	}
	if opts.Size > 0 {
		values.Set("size", fmt.Sprintf("%d", opts.Size))
	}
	if opts.Variant != "" {
		values.Set("variant", opts.Variant)
	}
	path := c.linkPath(spaceID, linkID) + "/qr/image"
	if len(values) > 0 {
		path += "?" + values.Encode()
	}
	return c.DoRaw(ctx, http.MethodGet, path)
}

// ExportQr materializes a design's stable public image URLs. An empty variantID
// exports the link's effective default design.
func (c *Client) ExportQr(ctx context.Context, spaceID string, linkID string, variantID string) (QrExportResponse, error) {
	body := map[string]string{}
	if variantID != "" {
		body["variant"] = variantID
	}
	var response QrExportResponse
	err := c.DoJSON(ctx, http.MethodPost, c.linkPath(spaceID, linkID)+"/qr/export", body, &response)
	return response, err
}

func (c *Client) linkPath(spaceID string, linkID string) string {
	return "/spaces/" + url.PathEscape(spaceID) + "/links/" + url.PathEscape(linkID)
}

func (c *Client) collectionPath(spaceID string, collectionID string) string {
	return "/spaces/" + url.PathEscape(spaceID) + "/collections/" + url.PathEscape(collectionID)
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
		return apiErrorFromResponse(res.StatusCode, method, path, data)
	}
	if out == nil || res.StatusCode == http.StatusNoContent {
		return nil
	}
	return json.Unmarshal(data, out)
}

// DoRaw performs a GET that returns non-JSON bytes (e.g. a rendered QR image),
// returning the body and its Content-Type. Non-2xx responses still carry a JSON
// error body, so they surface as *APIError just like DoJSON.
func (c *Client) DoRaw(ctx context.Context, method string, path string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, nil)
	if err != nil {
		return nil, "", err
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, "", apiErrorFromResponse(res.StatusCode, method, path, data)
	}
	return data, res.Header.Get("Content-Type"), nil
}

// apiErrorFromResponse builds an *APIError from a non-2xx body: the API's JSON
// error shape when present (keeping code/message/details), else the raw HTTP
// status with the body text as the message.
func apiErrorFromResponse(status int, method string, path string, data []byte) error {
	var parsed ErrorResponse
	if err := json.Unmarshal(data, &parsed); err == nil && parsed.Error.Message != "" {
		return &APIError{
			Status:  status,
			Code:    parsed.Error.Code,
			Message: parsed.Error.Message,
			Details: parsed.Error.Details,
		}
	}
	return &APIError{
		Status:  status,
		Message: fmt.Sprintf("HTTP %d from %s %s: %s", status, method, path, strings.TrimSpace(string(data))),
	}
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
	if options.IncludeClicks {
		values.Set("include", "clicks")
	}
	if len(values) == 0 {
		return ""
	}
	return "?" + values.Encode()
}
