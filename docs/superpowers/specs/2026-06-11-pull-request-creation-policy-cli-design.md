# Pull Request Creation Policy CLI Design

Date: 2026-06-11

Status: Approved design, pending implementation plan

## Goal

Add a `gh-usecase pull-request-creation-policy` command that configures a repository's GitHub pull request creation policy through the GitHub REST API.

Primary usage:

```bash
gh-usecase pull-request-creation-policy --owner y-writings --repo gh-usecase --policy collaborators_only
```

The command supports both GitHub API policy values:

- `all`
- `collaborators_only`

## Context

The repository is a Go CLI with command dispatch in `internal/cli` and GitHub API client construction in `internal/githubapi`. Existing repository-level mutating behavior is implemented by `codeql-default-setup`, which uses the `go-gh` REST client boundary and returns pretty-printed JSON on stdout.

GitHub exposes the setting on the Update a repository endpoint:

```txt
GET   /repos/{owner}/{repo}
PATCH /repos/{owner}/{repo}
```

The relevant repository response/body field is `pull_request_creation_policy`.

## Chosen Approach

Use a dedicated `internal/pullrequestcreationpolicy` package plus the existing REST client boundary in `internal/githubapi`.

The command first reads the current repository setting, compares it with the requested policy, and only sends `PATCH` when the repository is not already in the desired state. This keeps the command idempotent and mirrors the existing mutating command style.

## Scope

Included:

- Add the `pull-request-creation-policy` command.
- Require `--owner`, `--repo`, and `--policy`.
- Support exactly `all` and `collaborators_only` as policy values.
- Return stable JSON showing `before`, `after`, and whether a change was made.
- Use `go-gh` authentication and host resolution through the existing REST client.

Excluded:

- Do not add `--dry-run`.
- Do not add combined `owner/repo` input.
- Do not infer whether a repository is public, private, user-owned, or organization-owned.
- Do not modify any other repository settings.
- Do not add token flags or custom authentication handling.
- Do not introduce a generic repository settings framework in this phase.

## CLI Contract

Supported command:

```txt
gh-usecase pull-request-creation-policy --owner <owner> --repo <repo> --policy <all|collaborators_only>
```

Required options:

- `--owner <owner>`
- `--repo <repo>`
- `--policy <all|collaborators_only>`

Validation rules:

- `owner` must be non-empty.
- `repo` must be non-empty.
- `repo` must not contain `/`, so `--repo owner/name` is rejected before GitHub API calls.
- `policy` must be exactly `all` or `collaborators_only`.
- `policy` is case-sensitive.
- Unknown options are rejected before GitHub API calls.
- `--owner`, `--repo`, and `--policy` may each be specified only once.

Unsupported forms are rejected:

- `--repo owner/name`
- `--name` as an alias for `--repo`
- `--pull-request-creation-policy` as an alias for `--policy`
- `collaborators-only`, `COLLABORATORS_ONLY`, or other policy spellings
- Repeated allowed options such as `--policy all --policy collaborators_only`

## GitHub API Behavior

Data flow:

1. Parse `--owner`, `--repo`, and `--policy`.
2. Validate required options, enum values, unknown options, repeated options, and slash-containing repo values.
3. Construct the REST client.
4. GET `/repos/{owner}/{repo}`.
5. Read `pull_request_creation_policy` from the response.
6. If the current value matches the requested policy, return `changed: false` without PATCH.
7. If the current value differs, PATCH `/repos/{owner}/{repo}` with only `pull_request_creation_policy` in the body.
8. Return `changed: true` when PATCH succeeds.

PATCH body:

```json
{
  "pull_request_creation_policy": "collaborators_only"
}
```

The command does not re-GET after PATCH. The `after` field represents the desired policy submitted by the command, not a separately confirmed post-PATCH response.

If the GET response omits `pull_request_creation_policy`, the command treats the current value as an empty string. Because the empty value cannot equal a valid desired policy, the command attempts PATCH and lets GitHub validate the operation.

## Output Contract

Success behavior:

- stdout contains only pretty-printed JSON and a trailing newline.
- exit code is `0`.
- Field names use `snake_case` to match GitHub REST API naming.

Output shape:

```json
{
  "owner": "y-writings",
  "repo": "gh-usecase",
  "changed": true,
  "before": {
    "pull_request_creation_policy": "all"
  },
  "after": {
    "pull_request_creation_policy": "collaborators_only"
  }
}
```

When no PATCH is sent, `changed` is `false`, `before.pull_request_creation_policy` is the value returned by GitHub, and `after.pull_request_creation_policy` is the requested policy. In the unchanged case those values match.

Failure behavior:

- stderr contains human-readable text.
- exit code is `1`.
- Validation failures print command usage plus the reason.
- API, authentication, and transport failures print `Failed to execute command: ...`.
- No machine-readable error JSON is added.

## Error Handling

The command does not retry GitHub API failures.

These statuses are ordinary execution failures and are not interpreted specially by the command:

- `403`: missing permissions or repository policy restrictions.
- `404`: repository or API resource not found.
- `422`: invalid repository state or unsupported setting change.
- `503`: GitHub service unavailable.

Repository type, visibility, feature availability, and permission checks remain GitHub API responsibilities.

## Architecture

### `internal/pullrequestcreationpolicy`

Owns command behavior:

- Input validation.
- Policy enum validation.
- Repository path construction.
- Current policy GET.
- Desired policy construction.
- Difference detection.
- Conditional PATCH.
- Output construction.

This package depends only on the minimal `githubapi.RESTClient` interface.

### `internal/cli`

Adds command dispatch, usage text, argument validation glue, REST client construction, and JSON stdout encoding.

For this mutating command, unknown options and repeated allowed options are rejected before client creation.

### `internal/githubapi`

Reuse the existing REST client interface and `NewDefaultRESTClient()` implementation. No new authentication behavior is required.

### `README.md`

Update the command list and examples to include `pull-request-creation-policy`.

## Testing

Add offline unit tests. Tests must not require `GH_TOKEN`, network access, or live GitHub data.

`internal/pullrequestcreationpolicy` tests should cover:

- Required input validation.
- Policy enum acceptance for `all` and `collaborators_only`.
- Rejection of unknown and incorrectly cased policy values.
- Rejection of `repo` values containing `/`.
- GET path construction.
- No PATCH when current policy already matches desired policy.
- PATCH when current policy differs.
- PATCH body contains only `pull_request_creation_policy`.
- Missing current policy is treated as empty and causes PATCH.
- Output shape for changed and unchanged cases.
- API errors propagating without retry.

`internal/cli` tests should cover:

- Root usage lists `pull-request-creation-policy`.
- Command help output.
- Unknown option rejection for this command.
- Repeated `--owner`, `--repo`, and `--policy` rejection.
- Required option failures do not create a GitHub client.
- Successful command prints JSON.

Primary verification command:

```bash
go test ./...
```

## Non-Goals

- Do not add repository setting batching.
- Do not add a generic `repo-update` command.
- Do not add a common mutating-command framework until more duplication appears.
- Do not silently accept unsupported aliases or typoed options.
