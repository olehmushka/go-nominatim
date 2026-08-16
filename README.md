# go-nominatim

[![CI](https://github.com/olehmushka/go-nominatim/actions/workflows/ci.yml/badge.svg)](https://github.com/olehmushka/go-nominatim/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/olehmushka/go-nominatim.svg)](https://pkg.go.dev/github.com/olehmushka/go-nominatim)
[![Go Report Card](https://goreportcard.com/badge/github.com/olehmushka/go-nominatim)](https://goreportcard.com/report/github.com/olehmushka/go-nominatim)
[![License](https://img.shields.io/github/license/olehmushka/go-nominatim)](LICENSE)
[![Latest Release](https://img.shields.io/github/v/tag/olehmushka/go-nominatim)](https://github.com/olehmushka/go-nominatim/releases)

A small, rate-limited Go client for the public [OpenStreetMap Nominatim](https://nominatim.openstreetmap.org)
search API — free, keyless geocoding.

## Install

```sh
go get github.com/olehmushka/go-nominatim
```

## Usage

```go
c := nominatim.New(nil, nominatim.WithUserAgent("myapp/1.0 (contact@example.com)"))

result, found, err := c.Search(ctx, nominatim.Query{
    Street:     "Figueroa Alcorta y Oviedo",
    Locality:   "Villa María",
    AdminArea1: "Córdoba",
    Country:    "Argentina",
})
if err != nil {
    // real failure (network, non-200, bad response shape)
}
if !found {
    // real "no match" — not an error
}
```

## Notes from real-world use

- **Response shape**: `lat`/`lon` come back as JSON **strings**, not numbers — verified directly
  against the live endpoint, not assumed from the docs. This client parses them for you.
- **Reliability**: the public instance is a best-effort free community service, not a
  guaranteed-uptime API. Real connection resets/timeouts have been observed on repeated calls
  seconds apart from the same host. `New` defaults to a bounded 10s client timeout for this reason.
- **Usage policy**: Nominatim's own [usage policy](https://operations.osmfoundation.org/policies/nominatim/)
  caps general use at **1 request/second** and explicitly forbids bulk/systematic querying. This
  client enforces the rate limit internally with a real `rate.Limiter` — not just a comment — but it
  is still on the caller to only use this for one-off, human-triggered lookups, never as a bulk data
  source or crawler. Set `WithUserAgent` to identify your own application, as the policy requires.

## License

Apache 2.0 — see [LICENSE](./LICENSE).
