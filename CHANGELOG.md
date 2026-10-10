# Changelog
Format: Keep a Changelog; versioning: Semantic Versioning.

## [Unreleased]
### Changed
- CI, versioning and releases now follow the same pipeline as sweep-tracker. The version is the `VERSION` file; a merge to `main` publishes `latest`, `sha-<commit>` and the version tag (written once), and a pull request publishes a `pr-<number>` preview image that is deleted when it closes. The `staging` branch, the `:staging` image and the manual Release workflow are gone.
- The image is now built on distroless (non-root, no shell) for `linux/amd64` and `linux/arm64`, with a `healthcheck` subcommand for Docker's `HEALTHCHECK`.
- `GET /health` also reports the server version, and the binary has a `version` subcommand.
- `make test` runs `go vet` and the race detector; new `make cover` (80% gate), `make image` and `make smoke`.

### Added
- A smoke test that runs the built image against a real Postgres container before it is published.
