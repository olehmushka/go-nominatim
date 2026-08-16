# Contributing to go-nominatim

Thanks for your interest in improving go-nominatim.

## Development

Requires Go (the version pinned in [`go.mod`](go.mod)).

```sh
go build ./...
go vet ./...
go test ./...
gofmt -l .   # should print nothing; `gofmt -w .` fixes any hits
```

CI ([`.github/workflows/ci.yml`](.github/workflows/ci.yml)) runs the same build/vet/test steps on
every push and pull request.

## Submitting changes

1. Fork the repo and branch from `main`.
2. Keep changes focused; add or update tests for any behavior change.
3. Open a pull request describing what changed and why (see the PR template).

## Reporting bugs / requesting features

Use the issue templates. For security issues, see [SECURITY.md](SECURITY.md) instead of opening a
public issue.

## License

By contributing, you agree that your contributions are licensed under this project's
[Apache 2.0 License](LICENSE).
