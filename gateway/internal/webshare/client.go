package webshare

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultV2Base = "https://proxy.webshare.io/api/v2"
	defaultV3Base = "https://proxy.webshare.io/api/v3"
	maxPageSize   = 100
	maxPages      = 200
	maxBodyBytes  = 8 << 20
)

var (
	ErrUnauthorized  = errors.New("webshare unauthorized")
	ErrForbidden     = errors.New("webshare plan forbidden")
	ErrRateLimited   = errors.New("webshare rate limited")
	ErrUnavailable   = errors.New("webshare unavailable")
	ErrInvalidAnswer = errors.New("invalid webshare response")
)

// Error is a sanitized Webshare API error. It never contains the API key.
type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("webshare api status %d: %s", e.Status, e.Message)
	}
	return fmt.Sprintf("webshare api status %d", e.Status)
}

// Proxy is a direct proxy entry returned by the list endpoint.
type Proxy struct {
	ID             string `json:"id"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	Address        string `json:"proxy_address"`
	Port           int    `json:"port"`
	Valid          bool   `json:"valid"`
	LastVerifiedAt string `json:"last_verification"`
	CountryCode    string `json:"country_code"`
	CityName       string `json:"city_name"`
	CreatedAt      string `json:"created_at"`
}

// PlanConfig is the v2 plan configuration used to discover the plan id.
type PlanConfig struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

// Config is the plan configuration returned by v3.
type Config struct {
	PlanID                  string `json:"id"`
	ProxyListDownloadToken  string `json:"proxy_list_download_token"`
	OnDemandRefreshes       int    `json:"on_demand_refreshes_available"`
	AutoReplaceInProgress   bool   `json:"auto_replace_in_progress"`
	IPAuthorizationRequired bool   `json:"ip_authorization_required"`
}

// Status is the plan proxy status returned by v3.
type Status struct {
	State            string         `json:"state"`
	Countries        map[string]int `json:"countries"`
	Unallocated      map[string]int `json:"unallocated_countries"`
	Username         string         `json:"username"`
	Password         string         `json:"password"`
	IsProxyUsed      bool           `json:"is_proxy_used"`
	ProxyCount       int            `json:"proxy_count"`
	UnallocatedCount int            `json:"unallocated_count"`
}

// Stats is the plan proxy statistics returned by v3.
type Stats struct {
	AvailableCountries map[string]int `json:"available_countries"`
	ASNs               map[string]int `json:"asns"`
	AvailableASNs      map[string]int `json:"available_asns"`
}

// ListResult is one page of the proxy list.
type ListResult struct {
	Count    int     `json:"count"`
	Next     string  `json:"next"`
	Previous string  `json:"previous"`
	Results  []Proxy `json:"results"`
}

// Client talks to the Webshare API. The HTTP client is injected for tests.
type Client struct {
	HTTP   *http.Client
	V2Base string
	V3Base string
	Sleep  func(context.Context, time.Duration) error
}

// NewClient creates a Webshare client with production defaults.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	return &Client{
		HTTP:   httpClient,
		V2Base: defaultV2Base,
		V3Base: defaultV3Base,
		Sleep:  sleepContext,
	}
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// ListDirect returns every direct proxy for the plan, following pagination.
func (c *Client) ListDirect(ctx context.Context, apiKey, planID string) ([]Proxy, error) {
	proxies := make([]Proxy, 0)
	page := 1
	for page <= maxPages {
		query := url.Values{}
		query.Set("mode", "direct")
		query.Set("page", strconv.Itoa(page))
		query.Set("page_size", strconv.Itoa(maxPageSize))
		if planID != "" {
			query.Set("plan_id", planID)
		}
		var result ListResult
		if err := c.getJSON(ctx, apiKey, c.V2Base+"/proxy/list/?"+query.Encode(), &result); err != nil {
			return nil, err
		}
		proxies = append(proxies, result.Results...)
		if len(result.Results) == 0 || result.Next == "" || (result.Count > 0 && len(proxies) >= result.Count) {
			break
		}
		page++
	}
	return proxies, nil
}

// DiscoverPlanID reads the plan id from the v2 configuration endpoint.
func (c *Client) DiscoverPlanID(ctx context.Context, apiKey string) (string, error) {
	var config PlanConfig
	if err := c.getJSON(ctx, apiKey, c.V2Base+"/proxy/config/", &config); err != nil {
		return "", err
	}
	if config.ID <= 0 {
		return "", fmt.Errorf("%w: plan id is missing", ErrInvalidAnswer)
	}
	return strconv.Itoa(config.ID), nil
}

// Config returns the v3 plan configuration.
func (c *Client) Config(ctx context.Context, apiKey, planID string) (Config, error) {
	var config Config
	if err := c.getJSON(ctx, apiKey, c.V3Base+"/proxy/config"+planQuery(planID), &config); err != nil {
		return Config{}, err
	}
	return config, nil
}

// Status returns the v3 plan proxy status.
func (c *Client) Status(ctx context.Context, apiKey, planID string) (Status, error) {
	var status Status
	if err := c.getJSON(ctx, apiKey, c.V3Base+"/proxy/list/status"+planQuery(planID), &status); err != nil {
		return Status{}, err
	}
	return status, nil
}

// Stats returns the v3 plan proxy statistics.
func (c *Client) Stats(ctx context.Context, apiKey, planID string) (Stats, error) {
	var stats Stats
	if err := c.getJSON(ctx, apiKey, c.V3Base+"/proxy/list/stats"+planQuery(planID), &stats); err != nil {
		return Stats{}, err
	}
	return stats, nil
}

// Refresh requests an on-demand list refresh.
func (c *Client) Refresh(ctx context.Context, apiKey, planID string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.V2Base+"/proxy/list/refresh/"+planQuery(planID), nil)
	if err != nil {
		return fmt.Errorf("create webshare refresh request: %w", err)
	}
	return c.do(ctx, apiKey, request, nil)
}

// Download fetches the bulk proxy list for a download token.
func (c *Client) Download(ctx context.Context, token, countries, usernameMode, mode, search, planID string) (string, error) {
	segments := []string{
		url.PathEscape(token),
		url.PathEscape(defaultString(countries, "-")),
		"any",
		url.PathEscape(defaultString(usernameMode, "username")),
		url.PathEscape(defaultString(mode, "direct")),
		url.PathEscape(defaultString(search, "-")),
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.V2Base+"/proxy/list/download/"+strings.Join(segments, "/")+"/"+planQuery(planID), nil)
	if err != nil {
		return "", fmt.Errorf("create webshare download request: %w", err)
	}
	var body []byte
	if err := c.do(ctx, "", request, &body); err != nil {
		return "", err
	}
	return string(body), nil
}

func (c *Client) getJSON(ctx context.Context, apiKey, endpoint string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create webshare request: %w", err)
	}
	return c.do(ctx, apiKey, request, target)
}

func (c *Client) do(ctx context.Context, apiKey string, request *http.Request, target any) error {
	if apiKey != "" {
		request.Header.Set("Authorization", "Token "+apiKey)
	}
	request.Header.Set("Accept", "application/json")
	response, err := c.HTTP.Do(request)
	if err == nil && response.StatusCode == http.StatusTooManyRequests {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxBodyBytes))
		_ = response.Body.Close()
		if sleep := c.Sleep; sleep != nil {
			if sleepErr := sleep(ctx, 60*time.Second); sleepErr != nil {
				return sleepErr
			}
		}
		response, err = c.HTTP.Do(request.Clone(ctx))
	}
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxBodyBytes))
		return mapStatus(response.StatusCode)
	}
	if target == nil {
		return nil
	}
	if raw, ok := target.(*[]byte); ok {
		body, readErr := io.ReadAll(io.LimitReader(response.Body, maxBodyBytes))
		if readErr != nil {
			return fmt.Errorf("%w: read body", ErrInvalidAnswer)
		}
		*raw = body
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, maxBodyBytes)).Decode(target); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidAnswer, err)
	}
	return nil
}

func mapStatus(status int) error {
	switch {
	case status == http.StatusUnauthorized:
		return fmt.Errorf("%w: status 401", ErrUnauthorized)
	case status == http.StatusForbidden:
		return fmt.Errorf("%w: status 403", ErrForbidden)
	case status == http.StatusTooManyRequests:
		return fmt.Errorf("%w: status 429", ErrRateLimited)
	case status >= 500:
		return fmt.Errorf("%w: status %d", ErrUnavailable, status)
	default:
		return &Error{Status: status}
	}
}

func planQuery(planID string) string {
	if planID == "" {
		return ""
	}
	return "?plan_id=" + url.QueryEscape(planID)
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
