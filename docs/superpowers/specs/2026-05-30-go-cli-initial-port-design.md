# Go CLI Initial Port Design

Date: 2026-05-30

## Purpose

Rewrite the current Bun-based command implementation as an initial Go CLI port while preserving current command behavior. This phase establishes Go as the command behavior source of truth, but it does not implement npm binary distribution, TypeScript wrapper APIs, platform-specific npm packages, release automation, or removal of the Bun entrypoints.

## Scope

Included in this phase:

- Add a Go module at `github.com/y-writings/gh-usecase`.
- Pin the latest available Go `1.26.x` toolchain in `.mise/config.toml` during implementation.
- Add a Go CLI entrypoint under `cmd/gh-usecase`.
- Implement `pr-count`, `pr-list`, and `pr-detail` in Go.
- Use `github.com/cli/go-gh/v2/pkg/api.GraphQLClient` to call GitHub GraphQL directly.
- Preserve current CLI argument behavior and JSON output shape.
- Add shared JSON fixture/golden tests that can be reused across Bun and Go tests.
- Keep the current Bun implementation, package name, and `package.json` bin unchanged.

Excluded from this phase:

- npm package rename.
- TypeScript wrapper over the Go binary.
- platform-specific npm package layout.
- release automation.
- deleting or deprecating Bun entrypoints.
- changing `pr-detail` output to match the minimum-field rationale in `docs/pr-detail-field-rationale.md`.

## Approach

Use a parity-first Go CLI port with shared fixtures.

Before writing the Go command behavior, minimally refactor the Bun implementation so the transformation logic can be imported in tests without executing CLI entrypoints. The fixtures then lock current behavior, especially the `pr-detail` output transformation. The Go implementation uses the same fixtures with a fake GraphQL client to prove behavioral parity without hitting the network.

This avoids mixing behavior preservation with npm distribution work. It also avoids relying on manual comparisons or live GitHub API calls during the rewrite.

## Fixture And Golden Tests

Shared fixtures live under `testdata/`.

Recommended layout:

```txt
testdata/
  pr-count/
  pr-list/
  pr-detail/
```

Each command fixture should contain:

- GraphQL response input JSON.
- Expected final output JSON.
- Command input and expected GraphQL variables where useful.

The test strategy is:

- Refactor Bun code only enough to expose pure transformation boundaries for tests.
- Assert current Bun behavior against the shared fixtures.
- Assert Go behavior against the same fixtures through a fake `GraphQLClient`.
- Compare expected output semantically as JSON rather than relying on whitespace-sensitive snapshots.
- Keep default tests offline. Tests must not require `GH_TOKEN`, network access, rate-limit budget, or stable live GitHub data.

Primary parity targets:

- `pr-count`: output shape and `state` handling.
- `pr-list`: output shape, pagination args, default `first = 30`, optional `state`, optional `after`.
- `pr-detail`: nullable PR, review start anchor, generated/binary file exclusion, flattened authors, review/reviewThread/comment transforms, commit transforms.
- CLI validation: required params, enum values, positive integers, `1..100` bounds.

## Go Architecture

The Go implementation should use small responsibility boundaries without introducing a broad command framework.

Proposed layout:

```txt
cmd/gh-usecase/
  main.go

internal/cli/
  args.go
  runner.go
  usage.go

internal/githubapi/
  client.go

internal/prcount/
  prcount.go
  query.go
  types.go

internal/prlist/
  prlist.go
  query.go
  types.go

internal/prdetail/
  prdetail.go
  query.go
  types.go
  file_utils.go

testdata/
  pr-count/
  pr-list/
  pr-detail/
```

Responsibilities:

- `cmd/gh-usecase` handles process entrypoint, stdout/stderr, and exit code.
- `internal/cli` handles command dispatch, custom argument parsing, usage text, and JSON stdout encoding.
- `internal/githubapi` owns production `go-gh` client construction and the minimal GraphQL client interface.
- Each command package owns its query string, input validation, variable assembly, response structs, and output shaping.

Do not add a plugin system, generic command framework, compatibility shim, separate `internal/graphql` package, or separate `internal/jsonutil` package unless the implementation proves there is a concrete need.

## CLI Contract

The Go CLI binary name is `gh-usecase`. The current Bun/npm entrypoint remains unchanged during this phase.

Supported commands:

```txt
gh-usecase pr-count --owner <owner> --name <repo-name> [--state <OPEN|CLOSED|MERGED>]
gh-usecase pr-list --owner <owner> --name <repo-name> [--state <OPEN|CLOSED|MERGED>] [--after <cursor>] [--first <1-100>]
gh-usecase pr-detail --owner <owner> --name <repo-name> --number <pull-request-number> [--filesFirst <1-100>]
```

Argument parser behavior preserves the current Bun parser contract:

- Accept `--key value`.
- Accept `--key=value`.
- Accept `--help` and `-h`.
- Keep `--filesFirst`; do not add `--files-first` in this phase.
- Ignore unknown options and positional args.
- Validate known options per command.
- Default `first = 30` for `pr-list`.
- Default `filesFirst = 40` for `pr-detail`.

Go help text should describe `gh-usecase ...`, not `bun run ...`. Help output does not need byte-for-byte parity with Bun help.

## GitHub API Boundary

The production Go path calls GitHub GraphQL directly through `go-gh`; it does not spawn `gh` and does not implement GitHub auth manually.

Use this minimal interface so command behavior can be tested without network calls:

```go
type GraphQLClient interface {
    DoWithContext(ctx context.Context, query string, variables map[string]interface{}, response interface{}) error
}
```

Production behavior:

- Construct the client with `api.DefaultGraphQLClient()`.
- Execute requests with `DoWithContext`.
- Use `context.Background()` for initial CLI execution.
- Do not add a CLI-level timeout in this phase.
- Let `go-gh` handle `GH_TOKEN`, `GITHUB_TOKEN`, `GH_HOST`, and GitHub CLI config conventions.

Variable rules:

- `owner` and `name` are strings.
- `number`, `first`, and `filesFirst` are integers.
- Optional values are omitted from the variables map when not provided, except for command defaults such as `first = 30` and `filesFirst = 40`, which are always sent after validation/defaulting.
- `state` remains a single CLI value, but Go normalizes it to `[]string{state}` for the GraphQL variable type `[PullRequestState!]`.

## Output And Error Contract

Success behavior:

- stdout contains only JSON.
- JSON is pretty-printed with 2-space indentation and a trailing newline.
- exit code is `0`.
- Output shapes match current Bun behavior unless a future breaking-change decision changes them.

Failure behavior:

- stderr contains human-readable text.
- exit code is non-zero.
- Stderr text does not need to exactly match Bun or `gh` CLI stderr.
- Validation failures print usage plus a reason.
- GitHub/API failures preserve equivalent context to `Failed to execute command: ...`.
- No machine-readable error JSON is added in this phase.
- `go-gh` errors are not translated to mimic `gh api graphql` process semantics.

Validation and decoding rules:

- Known CLI inputs are validated explicitly.
- GraphQL responses decode into typed Go structs.
- Nullable GraphQL fields use pointers.
- Unknown GraphQL response fields are not rejected.
- Final output shape is fixed by golden fixtures.

## Migration Order

1. Add shared fixtures under `testdata/` for the three commands.
2. Refactor Bun code minimally so transformation logic is importable without invoking CLI entrypoints.
3. Add Bun-side fixture tests to lock current behavior.
4. Add Go module and pin latest available Go `1.26.x` in mise.
5. Add the Go CLI skeleton and command dispatch.
6. Add `internal/githubapi` and the minimal GraphQL client interface.
7. Port `pr-count` and verify fixture parity.
8. Port `pr-list` and verify fixture parity.
9. Port `pr-detail` and verify fixture parity.
10. Run offline verification for Bun tests, Go tests, lint/typecheck where available.
11. Run a manual smoke command against GitHub only as an optional verification step when credentials are available.

## Non-Goals

- Do not introduce a hidden pagination or aggregation loop.
- Do not add local filtering beyond existing transformation behavior.
- Do not change command output shapes.
- Do not solve the future npm-distributed tool's end-user runtime installation story in this phase.
- Do not require `gh` executable at runtime in the Go implementation.
- Do not add browser-compatible JavaScript APIs.
