# Contributing to Ares Bib Logger

## Branch model

```
main ← staging ← feature/your-description
```

| Branch | Purpose |
|--------|---------|
| `main` | Production-only; every merge produces a versioned release |
| `staging` | Integration branch; always deployable; the `staging` Docker image is rebuilt on every merge |
| `feature/*` | All development work; always branch from `staging` |

Use `fix/`, `feat/`, or `chore/` prefixes in place of `feature/` when appropriate.

## Getting started

See [Developer Setup](README.md#developer-setup) in the README for prerequisites and the `make dev` workflow.

## Making a change

### 1. Branch from `staging`

```bash
git checkout staging
git pull origin staging
git checkout -b feature/your-description
```

### 2. Develop and test locally

```bash
make test    # backend + frontend tests
make lint    # golangci-lint + ESLint
make fmt     # gofmt + Prettier
```

The pre-commit hook (installed by `make install`) runs `fmt` and `lint` automatically on every commit.

### 3. Open a PR targeting `staging`

- **Target `staging`, not `main`** — PRs to `main` are reserved for staging → main promotions by maintainers.
- CI (lint + test) must pass before merge.
- At least one review is required.

### 4. After merge

On merge to `staging`, CI re-runs and the `staging`-tagged Docker image (`ghcr.io/kbball/ares-bib-logger:staging`) is rebuilt automatically. This is the pre-production image for validation before a release.

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
- [ ] PR targets `staging` (not `main`)

## Releases (maintainers only)

Releases follow `<major>.<minor>` versioning (e.g. `1.3`).

### Promote staging → main

1. Open a PR from `staging` into `main`. CI must pass and the PR must be reviewed.
2. Merge the PR.

### Trigger the release

After merging, run the Release workflow from `main`:

**GitHub UI:** Actions → Release → Run workflow → select `main` → enter version (e.g. `1.3`, no `v` prefix) → Run.

**CLI:**
```bash
gh workflow run release.yml --ref main -f version=1.3
```

The workflow:
- Verifies it is running on `main`
- Runs lint and tests
- Creates git tag `v1.3`
- Builds a multi-arch image (`linux/amd64`, `linux/arm64`) and pushes `ghcr.io/kbball/ares-bib-logger:1.3` and `:latest` to GHCR

> Always trigger the release from `main`. Selecting any other branch will fail the workflow's branch check.
