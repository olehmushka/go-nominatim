# Security Policy

## Supported versions

Only the latest tagged release is supported. Please upgrade before reporting an issue that may
already be fixed.

## Reporting a vulnerability

Please do **not** open a public GitHub issue for security vulnerabilities.

- **Preferred**: use GitHub's private vulnerability reporting for this repository
  (the "Security" tab → "Report a vulnerability").
- **Alternative**: email olegamysk@gmail.com with details and, if possible, a proof of concept.

You should get an initial response within a few days.

## Scope notes

This package makes outbound HTTP requests to a Nominatim endpoint (the public
`nominatim.openstreetmap.org` instance by default, or a caller-supplied `BaseURL`). It does not
execute untrusted input, parse anything beyond a JSON response, or hold credentials/secrets — the
most likely real-world issue class here is a response-parsing bug (e.g. a malformed/hostile JSON
payload from a malicious or compromised base URL), not a classic injection/auth vulnerability.
