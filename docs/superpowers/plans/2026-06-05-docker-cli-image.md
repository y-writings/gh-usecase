# Docker CLI Image Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Docker image that runs the existing `gh-usecase` Go CLI and can call GitHub API-backed commands through container arguments.

**Architecture:** Use a multi-stage Docker build. The builder stage compiles `./cmd/gh-usecase` with the pinned Go toolchain, and the runtime stage uses a non-root distroless image containing only the compiled binary. Docker arguments are passed directly to the CLI through `ENTRYPOINT ["/gh-usecase"]`.

**Tech Stack:** Go 1.26.3, Docker multi-stage builds, `gcr.io/distroless/static-debian12:nonroot`, GitHub API auth via `GH_TOKEN` / `GITHUB_TOKEN` / `GH_HOST` environment variables.

---

## Files To Create Or Modify

- Create: `.dockerignore`
  - Responsibility: keep local metadata, secrets, generated state, and irrelevant docs/test fixtures out of the Docker build context.
- Create: `Dockerfile`
  - Responsibility: build the Go CLI binary and package it into a minimal non-root runtime image.

Do not commit changes unless the user explicitly requests a commit.

## Task 1: Baseline Verification

**Files:**
- Read only: existing Go source files

- [ ] **Step 1: Run existing Go tests**

Run:

```bash
go test ./...
```

Expected: command exits 0 and every package reports `ok`, for example:

```text
ok  github.com/y-writings/gh-usecase/internal/cli
ok  github.com/y-writings/gh-usecase/internal/githubapi
ok  github.com/y-writings/gh-usecase/internal/prcount
ok  github.com/y-writings/gh-usecase/internal/prdetail
ok  github.com/y-writings/gh-usecase/internal/prlist
```

If the package order differs, that is acceptable as long as the command exits 0 and no package fails.

## Task 2: Add Docker Build Context Ignore Rules

**Files:**
- Create: `.dockerignore`

- [ ] **Step 1: Create `.dockerignore`**

Create `.dockerignore` with exactly this content:

```gitignore
.git
.entire
.opencode
.gh
docs
testdata
.DS_Store
*.log
gh-usecase
```

These rules intentionally exclude `.gh` because GitHub CLI host files can contain local auth state, and the Dockerfile only needs `go.mod`, `go.sum`, `cmd/`, and `internal/`.

- [ ] **Step 2: Verify the ignore file exists**

Run:

```bash
test -f .dockerignore
```

Expected: command exits 0.

## Task 3: Add Dockerfile

**Files:**
- Create: `Dockerfile`

- [ ] **Step 1: Run Docker build before adding the Dockerfile**

Run:

```bash
docker build -t gh-usecase .
```

Expected: command fails because no `Dockerfile` exists yet. The error text may vary by Docker version, but it should indicate Docker cannot find or read a Dockerfile.

- [ ] **Step 2: Create `Dockerfile`**

Create `Dockerfile` with exactly this content:

```dockerfile
# syntax=docker/dockerfile:1

FROM golang:1.26.3-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/gh-usecase ./cmd/gh-usecase

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/gh-usecase /gh-usecase

USER nonroot:nonroot
ENTRYPOINT ["/gh-usecase"]
```

The build stage copies only dependency files first so module downloads can be cached separately from source changes. The runtime stage contains no shell and no Go toolchain.

- [ ] **Step 3: Verify Docker image builds**

Run:

```bash
docker build -t gh-usecase .
```

Expected: command exits 0 and ends with a successful image export/tag, including text similar to:

```text
naming to docker.io/library/gh-usecase:latest done
```

Docker output can differ by builder implementation, so exit code 0 is the authoritative result.

## Task 4: Verify Container CLI Behavior

**Files:**
- Read only: `Dockerfile`
- Read only: `.dockerignore`

- [ ] **Step 1: Verify help command through the container**

Run:

```bash
docker run --rm gh-usecase --help
```

Expected: command exits 0 and prints the root usage text:

```text
Usage: gh-usecase <command> [options]

Commands:
  pr-count    Fetch pull request total count
  pr-list     Fetch pull request list
  pr-detail   Fetch pull request detail for analysis
```

- [ ] **Step 2: Verify argument forwarding with command help**

Run:

```bash
docker run --rm gh-usecase pr-count --help
```

Expected: command exits 0 and prints:

```text
Usage: gh-usecase pr-count --owner <owner> --name <name> [--state OPEN|CLOSED|MERGED]

Fetch pull request total count.
```

- [ ] **Step 3: Optionally run a live GitHub API smoke test when `GH_TOKEN` is set**

Run only if the local shell has a valid token in `GH_TOKEN`:

```bash
docker run --rm -e GH_TOKEN gh-usecase pr-count --owner octokit --name rest.js --state OPEN
```

Expected: command exits 0 and prints JSON with a numeric total count:

```json
{
  "data": {
    "repository": {
      "pullRequests": {
        "totalCount": 0
      }
    }
  }
}
```

The exact `totalCount` value can differ. The important behavior is that the JSON shape matches and the command exits 0.

## Task 5: Final Verification

**Files:**
- Verify: `.dockerignore`
- Verify: `Dockerfile`

- [ ] **Step 1: Run Go tests again**

Run:

```bash
go test ./...
```

Expected: command exits 0.

- [ ] **Step 2: Rebuild the Docker image from final files**

Run:

```bash
docker build -t gh-usecase .
```

Expected: command exits 0.

- [ ] **Step 3: Run final container help check**

Run:

```bash
docker run --rm gh-usecase --help
```

Expected: command exits 0 and prints root usage.

- [ ] **Step 4: Inspect working tree**

Run:

```bash
git status --short
```

Expected: output includes only the intended new files unless there were pre-existing unrelated changes:

```text
?? .dockerignore
?? Dockerfile
?? docs/superpowers/plans/2026-06-05-docker-cli-image.md
?? docs/superpowers/specs/2026-06-05-docker-cli-image-design.md
```

If unrelated files appear, leave them untouched.
