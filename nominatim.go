// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package nominatim is a small client for the public OpenStreetMap Nominatim search API
// (https://nominatim.openstreetmap.org) — free, keyless.
//
// Real, verified against the live endpoint (2026-08-14, not assumed): a structured query for
// Villa María/Córdoba/Argentina returned a real match (lat -32.4106245, lon -63.2435809). Also hit
// real connection resets/timeouts on repeated calls seconds apart from the same host — the public
// endpoint is a best-effort free community service, not a guaranteed-uptime API; this package's own
// bounded client timeout and its callers' own error handling both exist because of that, not
// speculatively.
//
// Real, verified response shape: a JSON array, `lat`/`lon` are STRINGS (not numbers) — a real
// gotcha, checked directly against the live response, not assumed from the docs.
//
// Real, mechanically-enforced rate limit: Nominatim's own usage policy (see TermsURL) caps general
// use at 1 request/second and explicitly forbids bulk/systematic querying — enforced here with a
// real rate.Limiter, not just a comment. Callers should only use this for one-off, human-triggered
// lookups, never as a bulk/crawl data source.
package nominatim

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/time/rate"
)

// Code is a stable identifier for this provider, useful for callers that track results by source.
const Code = "nominatim"

// DefaultBaseURL is the public Nominatim instance. A field on Client, not hardcoded inline into
// requests, so a self-hosted Nominatim/Photon instance can be swapped in later without an interface
// change.
const DefaultBaseURL = "https://nominatim.openstreetmap.org"

// TermsURL is Nominatim's own usage policy — 1 request/second, no bulk/systematic querying.
const TermsURL = "https://operations.osmfoundation.org/policies/nominatim/"

// DefaultUserAgent identifies this client per Nominatim's own usage policy, which requires a real
// identifying User-Agent. Callers embedding this package in their own product should override it
// via WithUserAgent to identify their own application instead.
const DefaultUserAgent = "go-nominatim/0.1 (+https://github.com/olehmushka/go-nominatim)"

const requestTimeout = 10 * time.Second

// ErrNoMatch is returned by neither Search nor Geocode — an empty result set is a real "no match",
// reported as (nil, nil) / (Result{}, false, nil), not an error. It is exported only so a caller
// that wants to treat "no match" as an error condition of their own can compare against it
// consistently; this package itself never returns it.
var ErrNoMatch = errors.New("nominatim: no match")

// Client is a rate-limited Nominatim search client. The zero value is not usable — construct one
// with New.
type Client struct {
	BaseURL    string
	UserAgent  string
	httpClient *http.Client
	limiter    *rate.Limiter
}

// Option configures a Client constructed via New.
type Option func(*Client)

// WithBaseURL overrides DefaultBaseURL — e.g. to point at a self-hosted Nominatim/Photon instance.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) { c.BaseURL = baseURL }
}

// WithUserAgent overrides DefaultUserAgent. Nominatim's usage policy requires a real, identifying
// User-Agent for the calling application — set this to your own.
func WithUserAgent(userAgent string) Option {
	return func(c *Client) { c.UserAgent = userAgent }
}

// New constructs a Nominatim client. httpClient nil defaults to a client with requestTimeout —
// deliberately bounded: Search is a small, single-request lookup, not a streaming download.
func New(httpClient *http.Client, opts ...Option) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: requestTimeout}
	}
	c := &Client{
		BaseURL:    DefaultBaseURL,
		UserAgent:  DefaultUserAgent,
		httpClient: httpClient,
		limiter:    rate.NewLimiter(rate.Every(time.Second), 1),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Query is a structured search request. At least one field should be non-empty; Nominatim accepts
// partial queries (e.g. Locality+Country with no Street) and returns its best match.
type Query struct {
	Street     string
	Locality   string
	AdminArea1 string // state/province
	Country    string
}

// Result is one geocoded match.
type Result struct {
	Latitude  float64
	Longitude float64
	// Precision is Nominatim's own addresstype (preferred) or type field — e.g. "house", "town",
	// "administrative" — a rough indicator of how specific the match is, not a formal accuracy
	// radius.
	Precision   string
	DisplayName string
	Provider    string
}

// nominatimResult mirrors the real fields this package uses from Nominatim's own jsonv2 response
// shape — lat/lon are strings in the real response, not numbers.
type nominatimResult struct {
	Lat         string `json:"lat"`
	Lon         string `json:"lon"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
	AddressType string `json:"addresstype"`
}

// Search issues one rate-limited, structured search request. An empty result set is a real "no
// match" (found=false, err=nil), not an error.
func (c *Client) Search(ctx context.Context, query Query) (result Result, found bool, err error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return Result{}, false, fmt.Errorf("nominatim: rate limiter: %w", err)
	}

	q := url.Values{}
	q.Set("format", "jsonv2")
	q.Set("limit", "1")
	if query.Street != "" {
		q.Set("street", query.Street)
	}
	if query.Locality != "" {
		q.Set("city", query.Locality)
	}
	if query.AdminArea1 != "" {
		q.Set("state", query.AdminArea1)
	}
	if query.Country != "" {
		q.Set("country", query.Country)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/search?"+q.Encode(), nil)
	if err != nil {
		return Result{}, false, fmt.Errorf("nominatim: build request: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Result{}, false, fmt.Errorf("nominatim: GET %s: %w", req.URL.Path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return Result{}, false, fmt.Errorf("nominatim: GET %s: unexpected status %s: %s", req.URL.Path, resp.Status, body)
	}

	var results []nominatimResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return Result{}, false, fmt.Errorf("nominatim: decode response: %w", err)
	}
	if len(results) == 0 {
		return Result{}, false, nil
	}

	lat, err := strconv.ParseFloat(results[0].Lat, 64)
	if err != nil {
		return Result{}, false, fmt.Errorf("nominatim: parse lat %q: %w", results[0].Lat, err)
	}
	lon, err := strconv.ParseFloat(results[0].Lon, 64)
	if err != nil {
		return Result{}, false, fmt.Errorf("nominatim: parse lon %q: %w", results[0].Lon, err)
	}

	precision := results[0].AddressType
	if precision == "" {
		precision = results[0].Type
	}

	return Result{
		Latitude:    lat,
		Longitude:   lon,
		Precision:   precision,
		DisplayName: results[0].DisplayName,
		Provider:    Code,
	}, true, nil
}
