// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package nominatim

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestSearchRealMatch(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		if r.Header.Get("User-Agent") == "" {
			t.Error("no User-Agent header sent — Nominatim's own usage policy requires one")
		}
		w.Header().Set("Content-Type", "application/json")
		// Real response shape captured live against nominatim.openstreetmap.org (2026-08-14) for
		// Villa María, Córdoba, Argentina — lat/lon are strings, not numbers.
		_, _ = w.Write([]byte(`[{"lat":"-32.4106245","lon":"-63.2435809","display_name":"Villa María, Municipio de Villa María, Pedanía Villa María, Departamento General San Martín, Córdoba, X5900, Argentina","type":"administrative","addresstype":"town"}]`))
	}))
	defer srv.Close()

	c := New(nil, WithBaseURL(srv.URL))

	result, found, err := c.Search(context.Background(), Query{
		Street:     "Figueroa Alcorta y Oviedo",
		Locality:   "Villa María",
		AdminArea1: "Córdoba",
		Country:    "Argentina",
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if !found {
		t.Fatal("got found=false for a real match")
	}
	if result.Latitude != -32.4106245 || result.Longitude != -63.2435809 {
		t.Fatalf("got (%v, %v), want (-32.4106245, -63.2435809) — string lat/lon parsing broke", result.Latitude, result.Longitude)
	}
	if result.Provider != Code {
		t.Fatalf("Provider = %q, want %q", result.Provider, Code)
	}
	if result.Precision != "town" {
		t.Fatalf("Precision = %q, want \"town\"", result.Precision)
	}
	if result.DisplayName == "" {
		t.Fatal("DisplayName is empty — a caller has nothing to sanity-check the match against")
	}

	q, err := url.ParseQuery(gotQuery)
	if err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]string{
		"street":  "Figueroa Alcorta y Oviedo",
		"city":    "Villa María",
		"state":   "Córdoba",
		"country": "Argentina",
	} {
		if got := q.Get(k); got != want {
			t.Errorf("query param %s = %q, want %q — structured params, not a free-text q=", k, got, want)
		}
	}
}

func TestSearchNoMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := New(nil, WithBaseURL(srv.URL))

	result, found, err := c.Search(context.Background(), Query{Locality: "Nowhere At All"})
	if err != nil {
		t.Fatalf("Search should not error on a real empty match, got: %v", err)
	}
	if found {
		t.Fatalf("got found=true for an empty match: %+v", result)
	}
}

func TestSearchUpstreamFailurePassesThrough(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(nil, WithBaseURL(srv.URL))

	_, _, err := c.Search(context.Background(), Query{Locality: "Anywhere"})
	if err == nil {
		t.Fatal("a real 500 from the provider should be a real error, not silently treated as no-match")
	}
}

func TestWithUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := New(nil, WithBaseURL(srv.URL), WithUserAgent("myapp/1.0 (contact@example.com)"))
	if _, _, err := c.Search(context.Background(), Query{Locality: "Anywhere"}); err != nil {
		t.Fatal(err)
	}
	if gotUA != "myapp/1.0 (contact@example.com)" {
		t.Fatalf("User-Agent = %q, want the overridden value", gotUA)
	}
}
