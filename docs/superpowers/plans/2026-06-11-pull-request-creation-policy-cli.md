# Pull Request Creation Policy CLI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `gh-usecase pull-request-creation-policy` to idempotently configure `pull_request_creation_policy` through GitHub REST.

**Architecture:** Add a focused `internal/pullrequestcreationpolicy` package that owns validation, REST paths, GET/PATCH behavior, and output shaping. Wire it into `internal/cli` using the existing `githubapi.RESTClient` boundary and keep stdout/stderr behavior consistent with `codeql-default-setup`.

**Tech Stack:** Go 1.26.3, standard library `net/http`/`encoding/json`, existing `github.com/cli/go-gh/v2` REST client boundary, offline Go unit tests.

---

## Commit Policy For This Workspace

Do not run `git commit` while executing this plan unless the user explicitly grants commit permission. Each task ends with a diff/status checkpoint instead of a commit step.

## File Structure

- Create `internal/pullrequestcreationpolicy/types.go`: input/output DTOs and patch request shape for this command.
- Create `internal/pullrequestcreationpolicy/pullrequestcreationpolicy.go`: validation, repository REST path construction, idempotent GET/PATCH execution.
- Create `internal/pullrequestcreationpolicy/pullrequestcreationpolicy_test.go`: offline unit tests for validation, idempotency, PATCH body, missing current value, and API error propagation.
- Modify `internal/cli/usage.go`: root usage command list and command-specific usage text.
- Modify `internal/cli/runner.go`: import the new package, register the command, parse/validate arguments, call the package, encode JSON.
- Modify `internal/cli/runner_test.go`: CLI-level tests for help, validation before client creation, repeated options, unknown options, and JSON output.
- Modify `README.md`: command list and example.
- No changes required in `internal/githubapi/client.go`; reuse `RESTClient` and `NewDefaultRESTClient()`.

## Task 1: Domain Package Validation

**Files:**
- Create: `internal/pullrequestcreationpolicy/pullrequestcreationpolicy_test.go`
- Create: `internal/pullrequestcreationpolicy/types.go`
- Create: `internal/pullrequestcreationpolicy/pullrequestcreationpolicy.go`

- [ ] **Step 1: Write failing validation tests**

Create `internal/pullrequestcreationpolicy/pullrequestcreationpolicy_test.go` with:

```go
package pullrequestcreationpolicy

import "testing"

func TestValidateRejectsMissingRequiredFields(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input Input
		want  string
	}{
		{name: "owner", input: Input{Repo: "repo", Policy: "all"}, want: "owner is required"},
		{name: "repo", input: Input{Owner: "owner", Policy: "all"}, want: "repo is required"},
		{name: "policy", input: Input{Owner: "owner", Repo: "repo"}, want: "policy is required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Validate(tc.input)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("Validate error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestValidateAcceptsAllowedPolicies(t *testing.T) {
	for _, policy := range []string{"all", "collaborators_only"} {
		t.Run(policy, func(t *testing.T) {
			got, err := Validate(Input{Owner: "owner", Repo: "repo", Policy: policy})
			if err != nil {
				t.Fatalf("Validate returned error: %v", err)
			}
			if got != policy {
				t.Fatalf("Validate policy = %q, want %q", got, policy)
			}
		})
	}
}

func TestValidateRejectsUnknownAndIncorrectCasePolicies(t *testing.T) {
	for _, policy := range []string{"collaborators-only", "COLLABORATORS_ONLY", "owners_only"} {
		t.Run(policy, func(t *testing.T) {
			_, err := Validate(Input{Owner: "owner", Repo: "repo", Policy: policy})
			if err == nil || err.Error() != "policy must be all or collaborators_only" {
				t.Fatalf("Validate error = %v, want policy enum validation", err)
			}
		})
	}
}

func TestValidateRejectsRepoFullName(t *testing.T) {
	_, err := Validate(Input{Owner: "owner", Repo: "owner/repo", Policy: "all"})

	if err == nil || err.Error() != "repo must not contain /" {
		t.Fatalf("Validate error = %v, want repo slash validation", err)
	}
}
```

- [ ] **Step 2: Run validation tests and verify they fail**

Run:

```bash
go test ./internal/pullrequestcreationpolicy
```

Expected: FAIL because `Input` and `Validate` are undefined.

- [ ] **Step 3: Add minimal domain types**

Create `internal/pullrequestcreationpolicy/types.go` with:

```go
package pullrequestcreationpolicy

type Input struct {
	Owner  string
	Repo   string
	Policy string
}

type PolicyConfig struct {
	PullRequestCreationPolicy string `json:"pull_request_creation_policy"`
}

type Output struct {
	Owner   string       `json:"owner"`
	Repo    string       `json:"repo"`
	Changed bool         `json:"changed"`
	Before  PolicyConfig `json:"before"`
	After   PolicyConfig `json:"after"`
}

type patchRequest struct {
	PullRequestCreationPolicy string `json:"pull_request_creation_policy"`
}
```

- [ ] **Step 4: Add validation implementation**

Create `internal/pullrequestcreationpolicy/pullrequestcreationpolicy.go` with:

```go
package pullrequestcreationpolicy

import (
	"strings"

	"github.com/y-writings/gh-usecase/internal/validation"
)

func Validate(input Input) (string, error) {
	if input.Owner == "" {
		return "", validation.New("owner is required")
	}
	if input.Repo == "" {
		return "", validation.New("repo is required")
	}
	if strings.Contains(input.Repo, "/") {
		return "", validation.New("repo must not contain /")
	}
	if input.Policy == "" {
		return "", validation.New("policy is required")
	}
	if input.Policy != "all" && input.Policy != "collaborators_only" {
		return "", validation.New("policy must be all or collaborators_only")
	}
	return input.Policy, nil
}
```

- [ ] **Step 5: Run validation tests and verify they pass**

Run:

```bash
go test ./internal/pullrequestcreationpolicy
```

Expected: PASS.

- [ ] **Step 6: Check diff without committing**

Run:

```bash
git diff -- internal/pullrequestcreationpolicy
git status --short
```

Expected: only the new `internal/pullrequestcreationpolicy` files are shown for this task.

## Task 2: Domain Package REST Execution

**Files:**
- Modify: `internal/pullrequestcreationpolicy/pullrequestcreationpolicy_test.go`
- Modify: `internal/pullrequestcreationpolicy/pullrequestcreationpolicy.go`

- [ ] **Step 1: Add failing REST behavior tests**

First replace the import block in `internal/pullrequestcreationpolicy/pullrequestcreationpolicy_test.go` with:

```go
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
)
```

Append this code to `internal/pullrequestcreationpolicy/pullrequestcreationpolicy_test.go`:

```go
type restCall struct {
	Method string
	Path   string
	Body   string
}

type fakeRESTClient struct {
	currentJSON string
	errOnGet    error
	errOnPatch  error
	calls       []restCall
}

func (c *fakeRESTClient) DoWithContext(ctx context.Context, method string, path string, body io.Reader, response interface{}) error {
	var bodyText string
	if body != nil {
		bodyBytes, err := io.ReadAll(body)
		if err != nil {
			return err
		}
		bodyText = string(bodyBytes)
	}
	c.calls = append(c.calls, restCall{Method: method, Path: path, Body: bodyText})

	switch method {
	case http.MethodGet:
		if c.errOnGet != nil {
			return c.errOnGet
		}
		return json.Unmarshal([]byte(c.currentJSON), response)
	case http.MethodPatch:
		if c.errOnPatch != nil {
			return c.errOnPatch
		}
		return json.Unmarshal([]byte(`{"pull_request_creation_policy":"collaborators_only"}`), response)
	default:
		return fmt.Errorf("unexpected method %s", method)
	}
}

func TestExecuteDoesNotPatchWhenPolicyMatches(t *testing.T) {
	client := &fakeRESTClient{currentJSON: `{"pull_request_creation_policy":"all"}`}

	output, err := Execute(context.Background(), client, Input{Owner: "y-writings", Repo: "repo", Policy: "all"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if output.Changed {
		t.Fatal("Changed = true, want false")
	}
	if output.Before.PullRequestCreationPolicy != "all" {
		t.Fatalf("Before policy = %q, want all", output.Before.PullRequestCreationPolicy)
	}
	if output.After.PullRequestCreationPolicy != "all" {
		t.Fatalf("After policy = %q, want all", output.After.PullRequestCreationPolicy)
	}
	if len(client.calls) != 1 || client.calls[0].Method != http.MethodGet {
		t.Fatalf("calls = %#v, want only GET", client.calls)
	}
	if client.calls[0].Path != "repos/y-writings/repo" {
		t.Fatalf("GET path = %q, want repository path", client.calls[0].Path)
	}
}

func TestExecutePatchesWhenPolicyDiffers(t *testing.T) {
	client := &fakeRESTClient{currentJSON: `{"pull_request_creation_policy":"all"}`}

	output, err := Execute(context.Background(), client, Input{Owner: "owner", Repo: "repo", Policy: "collaborators_only"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if !output.Changed {
		t.Fatal("Changed = false, want true")
	}
	if output.Before.PullRequestCreationPolicy != "all" {
		t.Fatalf("Before policy = %q, want all", output.Before.PullRequestCreationPolicy)
	}
	if output.After.PullRequestCreationPolicy != "collaborators_only" {
		t.Fatalf("After policy = %q, want collaborators_only", output.After.PullRequestCreationPolicy)
	}
	if len(client.calls) != 2 || client.calls[1].Method != http.MethodPatch {
		t.Fatalf("calls = %#v, want GET then PATCH", client.calls)
	}
	wantBody := `{"pull_request_creation_policy":"collaborators_only"}`
	if client.calls[1].Body != wantBody {
		t.Fatalf("PATCH body = %s, want %s", client.calls[1].Body, wantBody)
	}
}

func TestExecutePatchesWhenCurrentPolicyIsMissing(t *testing.T) {
	client := &fakeRESTClient{currentJSON: `{}`}

	output, err := Execute(context.Background(), client, Input{Owner: "owner", Repo: "repo", Policy: "all"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if !output.Changed {
		t.Fatal("Changed = false, want true for missing current policy")
	}
	if output.Before.PullRequestCreationPolicy != "" {
		t.Fatalf("Before policy = %q, want empty string", output.Before.PullRequestCreationPolicy)
	}
	if len(client.calls) != 2 || client.calls[1].Method != http.MethodPatch {
		t.Fatalf("calls = %#v, want PATCH", client.calls)
	}
}

func TestExecutePropagatesAPIErrors(t *testing.T) {
	client := &fakeRESTClient{errOnGet: errors.New("github failed")}

	_, err := Execute(context.Background(), client, Input{Owner: "owner", Repo: "repo", Policy: "all"})
	if err == nil || err.Error() != "github failed" {
		t.Fatalf("Execute error = %v, want github failed", err)
	}
}
```

- [ ] **Step 2: Run REST behavior tests and verify they fail**

Run:

```bash
go test ./internal/pullrequestcreationpolicy
```

Expected: FAIL because `Execute` is undefined.

- [ ] **Step 3: Implement REST execution**

Replace `internal/pullrequestcreationpolicy/pullrequestcreationpolicy.go` with:

```go
package pullrequestcreationpolicy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/y-writings/gh-usecase/internal/githubapi"
	"github.com/y-writings/gh-usecase/internal/validation"
)

func Execute(ctx context.Context, client githubapi.RESTClient, input Input) (Output, error) {
	policy, err := Validate(input)
	if err != nil {
		return Output{}, err
	}

	path := repositoryPath(input.Owner, input.Repo)

	var current PolicyConfig
	if err := client.DoWithContext(ctx, http.MethodGet, path, nil, &current); err != nil {
		return Output{}, err
	}

	desired := PolicyConfig{PullRequestCreationPolicy: policy}
	output := Output{
		Owner:   input.Owner,
		Repo:    input.Repo,
		Changed: false,
		Before:  current,
		After:   desired,
	}

	if current.PullRequestCreationPolicy == policy {
		return output, nil
	}

	request := patchRequest{PullRequestCreationPolicy: policy}
	body, err := json.Marshal(request)
	if err != nil {
		return Output{}, err
	}

	var patched PolicyConfig
	if err := client.DoWithContext(ctx, http.MethodPatch, path, bytes.NewReader(body), &patched); err != nil {
		return Output{}, err
	}

	output.Changed = true
	return output, nil
}

func Validate(input Input) (string, error) {
	if input.Owner == "" {
		return "", validation.New("owner is required")
	}
	if input.Repo == "" {
		return "", validation.New("repo is required")
	}
	if strings.Contains(input.Repo, "/") {
		return "", validation.New("repo must not contain /")
	}
	if input.Policy == "" {
		return "", validation.New("policy is required")
	}
	if input.Policy != "all" && input.Policy != "collaborators_only" {
		return "", validation.New("policy must be all or collaborators_only")
	}
	return input.Policy, nil
}

func repositoryPath(owner string, repo string) string {
	return fmt.Sprintf(
		"repos/%s/%s",
		url.PathEscape(owner),
		url.PathEscape(repo),
	)
}
```

- [ ] **Step 4: Run package tests and verify they pass**

Run:

```bash
go test ./internal/pullrequestcreationpolicy
```

Expected: PASS.

- [ ] **Step 5: Check diff without committing**

Run:

```bash
git diff -- internal/pullrequestcreationpolicy
git status --short
```

Expected: new package tests and implementation are present; no unrelated files changed in this task.

## Task 3: CLI Registration And Behavior

**Files:**
- Modify: `internal/cli/runner_test.go`
- Modify: `internal/cli/usage.go`
- Modify: `internal/cli/runner.go`

- [ ] **Step 1: Add failing CLI tests**

Append this helper and tests to `internal/cli/runner_test.go`:

```go
type restCall struct {
	Method string
	Path   string
	Body   string
}

type pullRequestCreationPolicyRESTClient struct {
	current string
	calls   []restCall
}

func (c *pullRequestCreationPolicyRESTClient) DoWithContext(ctx context.Context, method string, path string, body io.Reader, response interface{}) error {
	var bodyText string
	if body != nil {
		bodyBytes, err := io.ReadAll(body)
		if err != nil {
			return err
		}
		bodyText = string(bodyBytes)
	}
	c.calls = append(c.calls, restCall{Method: method, Path: path, Body: bodyText})

	switch method {
	case http.MethodGet:
		payload := map[string]string{"pull_request_creation_policy": c.current}
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		return json.Unmarshal(encoded, response)
	case http.MethodPatch:
		payload := map[string]string{"pull_request_creation_policy": "collaborators_only"}
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		return json.Unmarshal(encoded, response)
	default:
		return errors.New("unexpected REST method")
	}
}

func TestRunPrintsPullRequestCreationPolicyUsageWhenHelpFollowsCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"pull-request-creation-policy", "--help"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "Usage: gh-usecase pull-request-creation-policy") {
		t.Fatalf("stdout = %q, want pull request creation policy usage", stdout.String())
	}
	if !strings.Contains(stdout.String(), "all|collaborators_only") {
		t.Fatalf("stdout = %q, want policy enum", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunRootUsageListsPullRequestCreationPolicy(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run(nil, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "pull-request-creation-policy") {
		t.Fatalf("stdout = %q, want root usage to list pull-request-creation-policy", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunRejectsPullRequestCreationPolicyUnknownOptionBeforeCallingClient(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	restClient := &errorRESTClient{err: errors.New("REST client must not be called")}

	code := runWithClientFactories(
		[]string{"pull-request-creation-policy", "--owner", "octo", "--repo", "repo", "--policy", "all", "--name", "repo"},
		&stdout,
		&stderr,
		func() (githubapi.GraphQLClient, error) { return nil, errors.New("GraphQL client must not be created") },
		func() (githubapi.RESTClient, error) { return restClient, nil },
	)

	if code == 0 {
		t.Fatal("Run exit code = 0, want non-zero")
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Usage: gh-usecase pull-request-creation-policy") {
		t.Fatalf("stderr = %q, want command usage", stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown option --name") {
		t.Fatalf("stderr = %q, want unknown option", stderr.String())
	}
	if restClient.called {
		t.Fatal("REST client was called for invalid input")
	}
}

func TestRunRejectsPullRequestCreationPolicyRepeatedOptionsBeforeCallingClient(t *testing.T) {
	for _, tc := range []struct {
		name string
		argv []string
		want string
	}{
		{name: "owner", argv: []string{"pull-request-creation-policy", "--owner", "octo", "--owner", "other", "--repo", "repo", "--policy", "all"}, want: "owner may be specified only once"},
		{name: "repo", argv: []string{"pull-request-creation-policy", "--owner", "octo", "--repo", "repo", "--repo", "other", "--policy", "all"}, want: "repo may be specified only once"},
		{name: "policy", argv: []string{"pull-request-creation-policy", "--owner", "octo", "--repo", "repo", "--policy", "all", "--policy", "collaborators_only"}, want: "policy may be specified only once"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			restClient := &errorRESTClient{err: errors.New("REST client must not be called")}

			code := runWithClientFactories(
				tc.argv,
				&stdout,
				&stderr,
				func() (githubapi.GraphQLClient, error) { return nil, errors.New("GraphQL client must not be created") },
				func() (githubapi.RESTClient, error) { return restClient, nil },
			)

			if code == 0 {
				t.Fatal("Run exit code = 0, want non-zero")
			}
			if !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tc.want)
			}
			if restClient.called {
				t.Fatal("REST client was called for invalid input")
			}
		})
	}
}

func TestRunRejectsPullRequestCreationPolicyMissingRequiredOptionsBeforeCallingClient(t *testing.T) {
	for _, tc := range []struct {
		name string
		argv []string
		want string
	}{
		{name: "owner", argv: []string{"pull-request-creation-policy", "--repo", "repo", "--policy", "all"}, want: "owner is required"},
		{name: "repo", argv: []string{"pull-request-creation-policy", "--owner", "octo", "--policy", "all"}, want: "repo is required"},
		{name: "policy", argv: []string{"pull-request-creation-policy", "--owner", "octo", "--repo", "repo"}, want: "policy is required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			restClient := &errorRESTClient{err: errors.New("REST client must not be called")}

			code := runWithClientFactories(
				tc.argv,
				&stdout,
				&stderr,
				func() (githubapi.GraphQLClient, error) { return nil, errors.New("GraphQL client must not be created") },
				func() (githubapi.RESTClient, error) { return restClient, nil },
			)

			if code == 0 {
				t.Fatal("Run exit code = 0, want non-zero")
			}
			if !strings.Contains(stderr.String(), "Usage: gh-usecase pull-request-creation-policy") {
				t.Fatalf("stderr = %q, want command usage", stderr.String())
			}
			if !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tc.want)
			}
			if restClient.called {
				t.Fatal("REST client was called for invalid input")
			}
		})
	}
}

func TestRunRejectsPullRequestCreationPolicyRepoFullNameBeforeCallingClient(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	restClient := &errorRESTClient{err: errors.New("REST client must not be called")}

	code := runWithClientFactories(
		[]string{"pull-request-creation-policy", "--owner", "octo", "--repo", "octo/repo", "--policy", "all"},
		&stdout,
		&stderr,
		func() (githubapi.GraphQLClient, error) { return nil, errors.New("GraphQL client must not be created") },
		func() (githubapi.RESTClient, error) { return restClient, nil },
	)

	if code == 0 {
		t.Fatal("Run exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "repo must not contain /") {
		t.Fatalf("stderr = %q, want repo slash error", stderr.String())
	}
	if restClient.called {
		t.Fatal("REST client was called for invalid input")
	}
}

func TestRunRejectsPullRequestCreationPolicyInvalidPolicyBeforeCallingClient(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	restClient := &errorRESTClient{err: errors.New("REST client must not be called")}

	code := runWithClientFactories(
		[]string{"pull-request-creation-policy", "--owner", "octo", "--repo", "repo", "--policy", "COLLABORATORS_ONLY"},
		&stdout,
		&stderr,
		func() (githubapi.GraphQLClient, error) { return nil, errors.New("GraphQL client must not be created") },
		func() (githubapi.RESTClient, error) { return restClient, nil },
	)

	if code == 0 {
		t.Fatal("Run exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "policy must be all or collaborators_only") {
		t.Fatalf("stderr = %q, want policy enum error", stderr.String())
	}
	if restClient.called {
		t.Fatal("REST client was called for invalid input")
	}
}

func TestRunPullRequestCreationPolicyPrintsJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	restClient := &pullRequestCreationPolicyRESTClient{current: "all"}

	code := runWithClientFactories(
		[]string{"pull-request-creation-policy", "--owner", "octo", "--repo", "repo", "--policy", "collaborators_only"},
		&stdout,
		&stderr,
		func() (githubapi.GraphQLClient, error) { return nil, errors.New("GraphQL client must not be created") },
		func() (githubapi.RESTClient, error) { return restClient, nil },
	)

	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if len(restClient.calls) != 2 {
		t.Fatalf("REST calls = %d, want 2", len(restClient.calls))
	}
	if !strings.Contains(stdout.String(), "\"changed\": true") {
		t.Fatalf("stdout = %q, want changed true JSON", stdout.String())
	}
	if !strings.Contains(stdout.String(), "\"pull_request_creation_policy\": \"collaborators_only\"") {
		t.Fatalf("stdout = %q, want desired policy", stdout.String())
	}
}
```

Also update the table in `TestRunPrintsCommandUsageForHelpWithoutCreatingGitHubClient` to include the new command:

```go
{command: "pull-request-creation-policy", usage: "Usage: gh-usecase pull-request-creation-policy"},
```

- [ ] **Step 2: Run CLI tests and verify they fail**

Run:

```bash
go test ./internal/cli
```

Expected: FAIL because the command is unknown and usage text is missing.

- [ ] **Step 3: Add command usage text**

Modify `internal/cli/usage.go` so `RootUsage` includes the new command:

```go
const RootUsage = `Usage: gh-usecase <command> [options]

Commands:
  pr-count                      Fetch pull request total count
  pr-list                       Fetch pull request list
  pr-detail                     Fetch pull request detail for analysis
  codeql-default-setup          Configure CodeQL default setup for a repository
  pull-request-creation-policy  Configure who can create pull requests for a repository`
```

Add this constant below `CodeQLDefaultSetupUsage`:

```go
const PullRequestCreationPolicyUsage = `Usage: gh-usecase pull-request-creation-policy --owner <owner> --repo <repo> --policy <all|collaborators_only>

Configure who can create pull requests for a repository.

Policy must be one of: all, collaborators_only.`
```

- [ ] **Step 4: Wire command dispatch and runner**

Modify `internal/cli/runner.go`.

Add the import:

```go
"github.com/y-writings/gh-usecase/internal/pullrequestcreationpolicy"
```

Add the switch case in `runWithClientFactories`:

```go
case "pull-request-creation-policy":
	return runPullRequestCreationPolicy(parsed, stdout, stderr, newRESTClient)
```

Add this function after `runCodeQLDefaultSetup`:

```go
func runPullRequestCreationPolicy(parsed ParsedArgs, stdout io.Writer, stderr io.Writer, newClient restClientFactory) int {
	if parsed.Help {
		fmt.Fprintln(stdout, PullRequestCreationPolicyUsage)
		return 0
	}

	if err := rejectUnsupportedOptions(parsed, []string{"owner", "repo", "policy"}); err != nil {
		printCommandError(stderr, PullRequestCreationPolicyUsage, err)
		return 1
	}
	for _, option := range []string{"owner", "repo", "policy"} {
		if parsed.OptionOccurrences[option] > 1 {
			printCommandError(stderr, PullRequestCreationPolicyUsage, validation.New(option+" may be specified only once"))
			return 1
		}
	}

	input := pullrequestcreationpolicy.Input{
		Owner:  parsed.Options["owner"],
		Repo:   parsed.Options["repo"],
		Policy: parsed.Options["policy"],
	}
	if _, err := pullrequestcreationpolicy.Validate(input); err != nil {
		printCommandError(stderr, PullRequestCreationPolicyUsage, err)
		return 1
	}

	client, err := newClient()
	if err != nil {
		printExecutionError(stderr, err)
		return 1
	}

	output, err := pullrequestcreationpolicy.Execute(context.Background(), client, input)
	if err != nil {
		printCommandError(stderr, PullRequestCreationPolicyUsage, err)
		return 1
	}

	encoded, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, string(encoded))
	return 0
}
```

Update `isKnownCommand`:

```go
case "pr-count", "pr-list", "pr-detail", "codeql-default-setup", "pull-request-creation-policy":
	return true
```

- [ ] **Step 5: Run CLI tests and verify they pass**

Run:

```bash
go test ./internal/cli
```

Expected: PASS.

- [ ] **Step 6: Run all tests and verify integration**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 7: Check diff without committing**

Run:

```bash
git diff -- internal/cli internal/pullrequestcreationpolicy
git status --short
```

Expected: only planned source/test files are changed in addition to the existing uncommitted spec and plan docs.

## Task 4: README And Final Verification

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Update README command list**

Modify the command list in `README.md` to include:

```md
- `pull-request-creation-policy` - configure who can create pull requests for a repository
```

- [ ] **Step 2: Update README examples**

Add this example to the examples block in `README.md`:

```sh
./gh-usecase pull-request-creation-policy --owner y-writings --repo gh-usecase --policy collaborators_only
```

- [ ] **Step 3: Run full test suite**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 4: Verify user-facing help manually**

Run:

```bash
go run ./cmd/gh-usecase --help
go run ./cmd/gh-usecase pull-request-creation-policy --help
```

Expected: root help lists `pull-request-creation-policy`; command help shows `--policy <all|collaborators_only>` and the policy enum line.

- [ ] **Step 5: Final diff/status review without committing**

Run:

```bash
git diff -- README.md internal/cli internal/pullrequestcreationpolicy docs/superpowers/specs/2026-06-11-pull-request-creation-policy-cli-design.md docs/superpowers/plans/2026-06-11-pull-request-creation-policy-cli.md
git status --short
```

Expected: only intended files changed. Do not commit unless the user explicitly grants commit permission.
