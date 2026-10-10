# Changelog
Format: Keep a Changelog; versioning: Semantic Versioning.

## [Unreleased]
### Changed
- CI, versioning and releases now follow the same pipeline as sweep-tracker. The version is the `VERSION` file; a merge to `main` publishes `latest`, `sha-<commit>` and the version tag (written once), and a pull request publishes a `pr-<number>` preview image that is deleted when it closes. The `staging` branch, the `:staging` image and the manual Release workflow are gone.
- The image is now built on distroless (non-root, no shell) for `linux/amd64` and `linux/arm64`, with a `healthcheck` subcommand for Docker's `HEALTHCHECK`.
- `GET /health` also reports the server version, and the binary has a `version` subcommand.
- `make test` runs `go vet` and the race detector; new `make cover` (80% gate), `make image` and `make smoke`.

### Added
- **Run behind a reverse proxy under a path:** set `BASE_PATH` (for example `/bibs`) when a proxy such as Caddy serves the app at `https://host/bibs/` and strips the prefix. The server tells the page its prefix through `<base href>`, so assets, API calls, the live stream, the logo and page routes all work under it. Unset, the app serves at `/` as before.
- A smoke test that runs the built image against a real Postgres container before it is published.
