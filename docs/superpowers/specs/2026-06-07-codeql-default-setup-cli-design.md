# CodeQL Default Setup CLI Design

Date: 2026-06-07

Status: Approved for implementation

## Goal

Add a `gh-usecase codeql-default-setup` command that configures GitHub CodeQL default setup for a single repository through the GitHub REST API.

The command CLI-automates the repository settings screen flow for CodeQL default setup. It does not create or modify advanced setup workflows.

Primary usage:

```bash
gh-usecase codeql-default-setup --owner y-writings --repo example --languages go,javascript-typescript
```

## Context

The repository already contains a Go CLI entrypoint at `cmd/gh-usecase` with hand-rolled argument parsing in `internal/cli` and GitHub API client construction in `internal/githubapi`. Existing commands call GitHub GraphQL through `go-gh` and return JSON on stdout.

The new CodeQL default setup API is REST-based:

```txt
GET   /repos/{owner}/{repo}/code-scanning/default-setup
PATCH /repos/{owner}/{repo}/code-scanning/default-setup
```

The command assumes the caller has already selected repositories where `codeql = true` in `https://github.com/y-writings/gh-terraform`. The command does not read or validate `gh-terraform` state.

## Chosen Approach

Use a dedicated `internal/codeqldefaultsetup` package plus a minimal REST client boundary in `internal/githubapi`.

The command first reads the current default setup configuration, compares it to the expected configuration, and only sends `PATCH` when the repository is not already in the desired state. This makes the command idempotent and avoids unnecessary GitHub validation runs.

## Scope

Included:

- Add the `codeql-default-setup` command.
- Require `--owner`, `--repo`, and `--languages`.
- Configure CodeQL default setup to a fixed expected state, with languages supplied by the caller.
- Return a stable JSON result showing `before`, `after`, whether a change was made, and any validation run returned by GitHub.
- Use `go-gh` authentication and host resolution.

Excluded:

- Do not execute or rerun PR CodeQL checks.
- Do not wait for CodeQL validation run completion.
- Do not create, modify, or inspect advanced setup GitHub Actions workflows.
- Do not modify branch protection or repository rulesets.
- Do not inspect `gh-terraform` to determine whether the repository should be targeted.
- Do not add token flags or custom authentication handling.
- Do not add `--dry-run` in this phase.

## CLI Contract

Supported command:

```txt
gh-usecase codeql-default-setup --owner <owner> --repo <repo> --languages <csv>
```

Required options:

- `--owner <owner>`
- `--repo <repo>`
- `--languages <csv>`

`owner` is validated only as a non-empty string. `repo` is validated as a non-empty string that must not contain `/`, so `--repo owner/name` is rejected. Repository existence, permissions, CodeQL availability, and all other naming rules are validated by GitHub API responses.

`--languages` accepts a comma-separated list. The command trims surrounding whitespace around each value, removes duplicates, and sorts values for comparison and output. Language comparison treats the list as a set.

Allowed language values are exactly:

- `actions`
- `c-cpp`
- `csharp`
- `go`
- `java-kotlin`
- `javascript-typescript`
- `python`
- `ruby`
- `swift`

Language values are case-sensitive. `go` is accepted; `Go` is rejected.

Unsupported forms are rejected for this command:

- Unknown options.
- `--name` as an alias for `--repo`.
- `--language` as an alias for `--languages`.
- Repeated `--languages` options.
- Combined `--repo owner/name` repository identifiers.

Help text should state that the command configures CodeQL default setup with `runner_type=standard`, `query_suite=default`, and `threat_model=remote`.

## Expected GitHub Configuration

The desired setup sent to GitHub is always:

```json
{
  "state": "configured",
  "languages": ["<caller-supplied languages>"],
  "runner_type": "standard",
  "query_suite": "default",
  "threat_model": "remote"
}
```

`runner_label` is not sent because `runner_type` is fixed to `standard`.

`schedule` is not sent because the GitHub REST API documents it on the GET response but not as a PATCH body field.

The command treats existing configured languages as an exact set. For example, current `go,python` and desired `go` is a difference and results in a PATCH to `go` only.

## Architecture

### `internal/githubapi`

Add a REST client interface next to the existing GraphQL client interface:

```go
type RESTClient interface {
    DoWithContext(ctx context.Context, method string, path string, body io.Reader, response interface{}) error
}
```

Add `NewDefaultRESTClient()` using `github.com/cli/go-gh/v2/pkg/api.NewRESTClient` with explicit REST headers:

```txt
Accept: application/vnd.github+json
X-GitHub-Api-Version: 2022-11-28
```

Authentication, host resolution, `GH_TOKEN`, `GITHUB_TOKEN`, `GH_HOST`, and GitHub CLI config compatibility remain delegated to `go-gh`.

### `internal/codeqldefaultsetup`

Owns the command behavior:

- Input validation.
- Language CSV normalization.
- Current configuration GET.
- Expected configuration construction.
- Difference detection.
- Conditional PATCH.
- Output construction.

This package depends only on the minimal `githubapi.RESTClient` interface, not directly on `go-gh`.

### `internal/cli`

Adds command dispatch, usage text, argument validation glue, REST client construction, and JSON stdout encoding.

For this mutating command, unknown options are rejected before client creation. Existing PR commands can keep their current permissive parser behavior.

## Data Flow

1. Parse `--owner`, `--repo`, and `--languages`.
2. Reject unsupported options, repeated `--languages`, missing required options, or invalid languages.
3. Construct the REST client.
4. GET `/repos/{owner}/{repo}/code-scanning/default-setup`.
5. Normalize the current configuration into the command's output shape.
6. Construct the desired configuration.
7. Compare the current and desired settings.
8. If they match, return `changed: false` without PATCH.
9. If they differ, PATCH the desired configuration.
10. Return `changed: true` and include `run_id` / `run_url` when GitHub provides them.

The command does not re-GET after PATCH. The `after` field represents the desired state the command submitted, not a confirmed post-validation state.

## Output Contract

Success behavior:

- stdout contains only pretty-printed JSON and a trailing newline.
- exit code is `0`.
- Field names use `snake_case` to match the REST API style.

Output shape:

```json
{
  "owner": "y-writings",
  "repo": "example",
  "changed": true,
  "before": {
    "state": "not-configured",
    "languages": [],
    "runner_type": null,
    "runner_label": null,
    "query_suite": "",
    "threat_model": "",
    "schedule": null,
    "updated_at": null
  },
  "after": {
    "state": "configured",
    "languages": ["go"],
    "runner_type": "standard",
    "runner_label": null,
    "query_suite": "default",
    "threat_model": "remote"
  },
  "run_id": 123456,
  "run_url": "https://github.com/..."
}
```

`run_id` and `run_url` are nullable. They are `null` when no PATCH is sent or when GitHub does not return those fields.

`before.languages` is always an array. For `not-configured` or missing language data, it is `[]`.

`before.query_suite` and `before.threat_model` preserve unknown GitHub values as strings. If GitHub omits them, they are empty strings.

Failure behavior:

- stderr contains human-readable text.
- exit code is `1`.
- Validation failures print command usage plus the reason.
- API, authentication, and transport failures print `Failed to execute command: ...`.
- No machine-readable error JSON is added.

## Error Handling

The command does not retry GitHub API failures.

These statuses are ordinary failures:

- `403`: archived repository, missing permissions, or code scanning unavailable.
- `404`: repository or API resource not found.
- `409`: another validation run is already in progress with a different configuration.
- `422`: configuration change cannot be made because the repository is not in the required state.
- `503`: GitHub service unavailable.

`PATCH` success accepts both `200 OK` and `202 Accepted` because GitHub documents both status codes for the endpoint. `go-gh` treats any 2xx status as success.

## Testing

Add offline unit tests. Tests must not require `GH_TOKEN`, network access, or live GitHub data.

`internal/codeqldefaultsetup` tests should cover:

- Required input validation.
- Accepted language values.
- Rejection of unknown and incorrectly cased language values.
- CSV trimming, duplicate removal, and sorted output.
- Current and desired language set comparison ignoring order.
- GET path construction.
- PATCH path and body construction.
- No PATCH when current configuration already matches desired configuration.
- PATCH when state or any setting differs.
- Exact language set convergence, including reducing a superset.
- Output shape for changed and unchanged cases.
- Nullable `run_id` and `run_url`.
- API errors propagating without retry.

`internal/cli` tests should cover:

- Root usage lists `codeql-default-setup`.
- Command help output.
- Unknown option rejection for this command.
- Repeated `--languages` rejection.
- Required option failures do not create a GitHub client.
- Successful command prints JSON.

`internal/githubapi` should keep production REST client construction small enough to verify by inspection or a focused unit test around explicit headers if practical.

Primary verification command:

```bash
go test ./...
```

## Operational Notes

This command should usually run before the normal auto-merge PR workflow begins for a repository. CodeQL default setup is repository configuration, not a PR-level trigger. Running it after a PR is created may not guarantee the PR receives the required CodeQL check in time for auto-merge.

Bot workflows should treat this command as a repository setup step. PR check waiting, reruns, approvals, auto-merge enablement, and required-check configuration remain separate responsibilities.

## Non-Goals

- Do not call GitHub Actions workflow dispatch APIs.
- Do not call CodeQL analysis or SARIF upload APIs.
- Do not infer languages from repository contents.
- Do not silently accept unsupported aliases or typoed options.
- Do not introduce a general CodeQL package until another CodeQL command exists.
