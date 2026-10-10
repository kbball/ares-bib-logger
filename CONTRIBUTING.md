# Contributing to Ares Bib Logger

## Branch model

`main` is protected: nothing is committed to it directly. All work happens on a branch and reaches `main` through a pull request whose required checks (`Go`, `Frontend`, `Image`) pass.

Name branches `feature/…`, `fix/…` or `chore/…`, always cut from `main`.

## Getting started

See [Developer Setup](README.md#developer-setup) in the README for prerequisites and the `make dev` workflow.

## Making a change

### 1. Branch from `main`

```bash
git checkout main
git pull origin main
git checkout -b feature/your-description
```

### 2. Develop and test locally

```bash
make test    # vet + race detector, backend and frontend
make cover   # the coverage gate CI enforces (80%)
make lint    # golangci-lint + ESLint
make fmt     # gofmt + Prettier
make smoke   # build the image and run it against real Postgres
```

The pre-commit hook (installed by `make install`) runs `fmt` and `lint` automatically on every commit.

### 3. Open a pull request

- Target `main`. CI must pass and at least one review is required.
- Add an entry under `## [Unreleased]` in `CHANGELOG.md` for any user-visible change.
- A pull request from this repository publishes a **preview image** you can try before merging: `ghcr.io/kbball/ares-bib-logger:pr-<number>` (and `:<version>-pr<number>.<commit>`). It never moves `latest`. Previews are deleted when the PR closes.

## Commit messages

Short, imperative, with a type prefix:

| Prefix | When to use |
|--------|-------------|
| `feat:` | New capability |
| `fix:` | Bug fix |
| `test:` | Tests only |
| `refactor:` | No behavior change |
| `docs:` | Documentation only |
| `ci:` | CI/CD configuration |
| `chore:` | Maintenance, dependency updates |

Example: `feat: add pace projection to runner detail panel`

## Coding standards

See [CLAUDE.md](CLAUDE.md) for the full rule set. Short version:

- **Hexagonal architecture** — `domain/` and `application/` layers have zero framework imports; all infra lives in `adapter/`
- **Tests required** — every package must have tests; target >90% backend coverage, >80% frontend coverage
- **12-factor config** — all runtime values via environment variables; no hardcoded values
- **`log/slog` only** — no third-party logging libraries

## Pull request checklist

- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] New code has test coverage
- [ ] No hardcoded config values
- [ ] `CHANGELOG.md` updated for user-visible changes
- [ ] `make cover` passes

## Releases (maintainers only)

The version lives in the `VERSION` file (semantic versioning, `1.4.0`) and is the single source: the Makefile, the Docker `VERSION` build argument and the binary's reported version all read it.

To release:

1. In a pull request, bump `VERSION` and move `## [Unreleased]` in `CHANGELOG.md` into a dated `## [x.y.z]` section.
2. Merge it. CI on `main` builds the multi-arch image (`linux/amd64`, `linux/arm64`), runs the smoke test, and publishes `:latest`, `:sha-<commit>` and `:<VERSION>` to GHCR.

The version tag is written once and never replaced: if `:<VERSION>` already exists the run warns and skips it, so bump `VERSION` to publish a new version. Merging without a bump only moves `latest` and `sha-<commit>`.
