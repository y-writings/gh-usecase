# Docker CLI Image Design

Date: 2026-06-05

Status: Approved for implementation

## Goal

Add Docker support so users can run the existing `gh-usecase` CLI from a container and use it to call GitHub GraphQL API-backed commands.

The primary usage should be:

```bash
docker build -t gh-usecase .
docker run --rm -e GH_TOKEN gh-usecase pr-count --owner octokit --name rest.js --state OPEN
```

## Context

`gh-usecase` is currently a Go CLI, not an HTTP server. It exposes `pr-count`, `pr-list`, and `pr-detail` commands under `cmd/gh-usecase` and uses `github.com/cli/go-gh/v2/pkg/api` to call GitHub GraphQL APIs directly.

Authentication and host configuration remain delegated to `go-gh`, which supports `GH_TOKEN`, `GITHUB_TOKEN`, `GH_HOST`, and compatible GitHub CLI configuration conventions. Container usage should document environment-variable based auth as the standard path.

## Chosen Approach

Use a multi-stage Dockerfile at the repository root.

The build stage uses the Go toolchain to download modules and compile `./cmd/gh-usecase` into a standalone binary. The runtime stage uses a small non-root distroless image and copies only the compiled binary into it.

Entrypoint is the binary itself:

```dockerfile
ENTRYPOINT ["/gh-usecase"]
```

This preserves the current CLI contract. Docker arguments map directly to `gh-usecase` arguments, so no wrapper script or new command layer is needed.

## Files

- Add `Dockerfile` at the repository root.
- Add `.dockerignore` at the repository root to keep build context small and avoid copying local metadata.

## Behavior

- `docker run --rm gh-usecase --help` prints root CLI usage.
- `docker run --rm -e GH_TOKEN gh-usecase pr-count ...` calls the GitHub API through the existing CLI path.
- `GH_TOKEN`, `GITHUB_TOKEN`, and `GH_HOST` can be passed through with `-e` as needed.
- The image does not expose ports and does not start an HTTP server.

## Error Handling

The container should not add custom error handling. CLI validation errors, GitHub API errors, and auth errors continue to be printed by the existing Go CLI to stderr with its existing exit codes.

## Testing

Implementation verification should include:

- `go test ./...`
- `docker build -t gh-usecase .`
- `docker run --rm gh-usecase --help`

A live GitHub API smoke test can be run only when a valid token is available:

```bash
docker run --rm -e GH_TOKEN gh-usecase pr-count --owner octokit --name rest.js --state OPEN
```

## Non-Goals

- Do not add an HTTP server.
- Do not change CLI commands or JSON output shapes.
- Do not require `gh` executable in the runtime image.
- Do not add Docker Compose unless a later workflow needs it.
