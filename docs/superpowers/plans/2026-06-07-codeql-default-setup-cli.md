# CodeQL Default Setup CLI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `gh-usecase codeql-default-setup` to idempotently configure GitHub CodeQL default setup for one repository through the GitHub REST API.

**Architecture:** Add a minimal REST client boundary in `internal/githubapi`, a focused `internal/codeqldefaultsetup` package for validation, normalization, GET/PATCH, and output shaping, then wire a thin CLI command in `internal/cli`. The command reads current setup first, compares against the fixed desired setup, and only PATCHes when different.

**Tech Stack:** Go 1.26.3, `github.com/cli/go-gh/v2/pkg/api`, standard library `encoding/json`, `net/http`, `io`, offline Go unit tests.

---

## Files To Create Or Modify

- Modify: `internal/cli/args.go` to track option occurrences for duplicate and unknown option checks.
- Modify: `internal/cli/args_test.go` to test option occurrence tracking.
- Modify: `internal/githubapi/client.go` to add a REST client interface and default REST client factory.
- Modify: `internal/githubapi/client_test.go` to compile-check the REST client boundary.
- Create: `internal/codeqldefaultsetup/types.go` for input, current config, desired config, patch response, and output structs.
- Create: `internal/codeqldefaultsetup/codeqldefaultsetup.go` for validation, language normalization, comparison, and API execution.
- Create: `internal/codeqldefaultsetup/codeqldefaultsetup_test.go` for validation and idempotent GET/PATCH behavior.
- Modify: `internal/cli/usage.go` to add `codeql-default-setup` usage and root command listing.
- Modify: `internal/cli/runner.go` to dispatch the new command and use both GraphQL and REST client factories.
- Modify: `internal/cli/runner_test.go` to test command wiring, validation, help, and JSON output.

## Implementation Notes

- Use `repos/{owner}/{repo}/code-scanning/default-setup` without a leading slash because `go-gh` REST paths are joined to a prefix that already ends in `/`.
- Use `url.PathEscape` for both `owner` and `repo` when building the REST path.
- Reject `repo` values containing `/` before path construction, so `--repo owner/name` does not silently target an escaped invalid repository name.
- Compare only fields that the command can set: `state`, `languages`, `runner_type`, `query_suite`, and `threat_model`.
- Do not compare `runner_label`, `schedule`, or `updated_at`; they are included in `before` output but not part of the PATCH request.
- Keep existing PR commands permissive about unknown options. Strict unknown option rejection applies only to `codeql-default-setup`.

---

## Task 1: Track CLI Option Occurrences

**Files:**
- Modify: `internal/cli/args.go`
- Modify: `internal/cli/args_test.go`

- [ ] **Step 1: Add failing parser tests for option occurrences**

Append these tests to `internal/cli/args_test.go`:

```go
func TestParseArgsTracksOptionOccurrences(t *testing.T) {
	parsed := ParseArgs([]string{
		"codeql-default-setup",
		"--owner", "octo",
		"--repo=hello",
		"--languages", "go",
		"--languages", "python",
		"--flag-without-value",
	})

	wantOccurrences := map[string]int{
		"owner":              1,
		"repo":               1,
		"languages":          2,
		"flag-without-value": 1,
	}
	if !reflect.DeepEqual(parsed.OptionOccurrences, wantOccurrences) {
		t.Fatalf("OptionOccurrences = %#v, want %#v", parsed.OptionOccurrences, wantOccurrences)
	}

	if parsed.Options["languages"] != "python" {
		t.Fatalf("Options[languages] = %q, want last value python", parsed.Options["languages"])
	}
	if _, ok := parsed.Options["flag-without-value"]; ok {
		t.Fatalf("Options[flag-without-value] = %q, want missing", parsed.Options["flag-without-value"])
	}
}

func TestParseArgsTracksMissingOptionValueOccurrence(t *testing.T) {
	parsed := ParseArgs([]string{"codeql-default-setup", "--owner"})

	if parsed.OptionOccurrences["owner"] != 1 {
		t.Fatalf("OptionOccurrences[owner] = %d, want 1", parsed.OptionOccurrences["owner"])
	}
	if _, ok := parsed.Options["owner"]; ok {
		t.Fatalf("Options[owner] = %q, want missing", parsed.Options["owner"])
	}
}
```

- [ ] **Step 2: Run parser tests to verify they fail**

Run:

```bash
go test ./internal/cli -run 'TestParseArgsTracks' -count=1
```

Expected: FAIL with a compile error because `ParsedArgs.OptionOccurrences` is not defined.

- [ ] **Step 3: Implement option occurrence tracking**

Update `internal/cli/args.go` so it contains this complete implementation:

```go
package cli

import "strings"

type ParsedArgs struct {
	Options           map[string]string
	OptionOccurrences map[string]int
	Positionals       []string
	Help              bool
}

func ParseArgs(argv []string) ParsedArgs {
	parsed := ParsedArgs{
		Options:           make(map[string]string),
		OptionOccurrences: make(map[string]int),
	}

	for i := 0; i < len(argv); i++ {
		token := argv[i]

		if token == "--help" || token == "-h" {
			parsed.Help = true
			continue
		}

		if strings.HasPrefix(token, "--") {
			withoutPrefix := strings.TrimPrefix(token, "--")
			if withoutPrefix == "" {
				continue
			}

			if key, value, ok := strings.Cut(withoutPrefix, "="); ok {
				if key != "" {
					parsed.OptionOccurrences[key]++
					parsed.Options[key] = value
				}
				continue
			}

			parsed.OptionOccurrences[withoutPrefix]++
			if i+1 < len(argv) && !strings.HasPrefix(argv[i+1], "--") {
				parsed.Options[withoutPrefix] = argv[i+1]
				i++
			}

			continue
		}

		parsed.Positionals = append(parsed.Positionals, token)
	}

	return parsed
}
```

- [ ] **Step 4: Run parser tests to verify they pass**

Run:

```bash
go test ./internal/cli -run 'TestParseArgs' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit Task 1**

```bash
git add internal/cli/args.go internal/cli/args_test.go
git commit -m "feat: track cli option occurrences"
```

---

## Task 2: Add REST Client Boundary

**Files:**
- Modify: `internal/githubapi/client.go`
- Modify: `internal/githubapi/client_test.go`

- [ ] **Step 1: Add failing compile-boundary test**

Replace `internal/githubapi/client_test.go` with:

```go
package githubapi

import (
	"context"
	"io"
	"testing"
)

type compileOnlyGraphQLClient struct{}

func (compileOnlyGraphQLClient) DoWithContext(ctx context.Context, query string, variables map[string]interface{}, response interface{}) error {
	return nil
}

type compileOnlyRESTClient struct{}

func (compileOnlyRESTClient) DoWithContext(ctx context.Context, method string, path string, body io.Reader, response interface{}) error {
	return nil
}

func TestGraphQLClientBoundaryCompiles(t *testing.T) {
	var _ GraphQLClient = compileOnlyGraphQLClient{}
	var _ func() (GraphQLClient, error) = NewDefaultGraphQLClient
}

func TestRESTClientBoundaryCompiles(t *testing.T) {
	var _ RESTClient = compileOnlyRESTClient{}
	var _ func() (RESTClient, error) = NewDefaultRESTClient
}
```

- [ ] **Step 2: Run githubapi tests to verify they fail**

Run:

```bash
go test ./internal/githubapi -count=1
```

Expected: FAIL with compile errors because `RESTClient` and `NewDefaultRESTClient` are not defined.

- [ ] **Step 3: Implement REST client boundary**

Replace `internal/githubapi/client.go` with:

```go
package githubapi

import (
	"context"
	"io"

	"github.com/cli/go-gh/v2/pkg/api"
)

const restAPIVersion = "2022-11-28"

type GraphQLClient interface {
	DoWithContext(ctx context.Context, query string, variables map[string]interface{}, response interface{}) error
}

type RESTClient interface {
	DoWithContext(ctx context.Context, method string, path string, body io.Reader, response interface{}) error
}

func NewDefaultGraphQLClient() (GraphQLClient, error) {
	return api.DefaultGraphQLClient()
}

func NewDefaultRESTClient() (RESTClient, error) {
	return api.NewRESTClient(api.ClientOptions{
		Headers: map[string]string{
			"Accept":               "application/vnd.github+json",
			"X-GitHub-Api-Version": restAPIVersion,
		},
	})
}
```

- [ ] **Step 4: Run githubapi tests to verify they pass**

Run:

```bash
go test ./internal/githubapi -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit Task 2**

```bash
git add internal/githubapi/client.go internal/githubapi/client_test.go
git commit -m "feat: add github rest client boundary"
```

---

## Task 3: Add CodeQL Default Setup Validation

**Files:**
- Create: `internal/codeqldefaultsetup/types.go`
- Create: `internal/codeqldefaultsetup/codeqldefaultsetup.go`
- Create: `internal/codeqldefaultsetup/codeqldefaultsetup_test.go`

- [ ] **Step 1: Write failing validation and normalization tests**

Create `internal/codeqldefaultsetup/codeqldefaultsetup_test.go` with:

```go
package codeqldefaultsetup

import (
	"reflect"
	"strings"
	"testing"
)

func TestValidateRejectsMissingRequiredFields(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input Input
		want  string
	}{
		{name: "owner", input: Input{Repo: "repo", Languages: "go"}, want: "owner is required"},
		{name: "repo", input: Input{Owner: "owner", Languages: "go"}, want: "repo is required"},
		{name: "languages", input: Input{Owner: "owner", Repo: "repo"}, want: "languages is required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Validate(tc.input)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("Validate error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestValidateRejectsRepoFullName(t *testing.T) {
	_, err := Validate(Input{Owner: "owner", Repo: "owner/repo", Languages: "go"})

	if err == nil || err.Error() != "repo must not contain /" {
		t.Fatalf("Validate error = %v, want repo slash validation", err)
	}
}

func TestNormalizeLanguagesTrimsDeduplicatesAndSorts(t *testing.T) {
	got, err := NormalizeLanguages("go, javascript-typescript,go,python")
	if err != nil {
		t.Fatalf("NormalizeLanguages returned error: %v", err)
	}

	want := []string{"go", "javascript-typescript", "python"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeLanguages = %#v, want %#v", got, want)
	}
}

func TestNormalizeLanguagesRejectsUnknownAndIncorrectCase(t *testing.T) {
	for _, raw := range []string{"Go", "typescript", "go,"} {
		t.Run(raw, func(t *testing.T) {
			_, err := NormalizeLanguages(raw)
			if err == nil {
				t.Fatal("NormalizeLanguages returned nil error")
			}
			if !strings.Contains(err.Error(), "languages must contain only") && !strings.Contains(err.Error(), "languages must not contain empty values") {
				t.Fatalf("NormalizeLanguages error = %q, want language validation", err.Error())
			}
		})
	}
}
```

- [ ] **Step 2: Run validation tests to verify they fail**

Run:

```bash
go test ./internal/codeqldefaultsetup -run 'TestValidate|TestNormalize' -count=1
```

Expected: FAIL because the package and functions do not exist yet.

- [ ] **Step 3: Add types**

Create `internal/codeqldefaultsetup/types.go`:

```go
package codeqldefaultsetup

type Input struct {
	Owner     string
	Repo      string
	Languages string
}

type CurrentConfig struct {
	State       string   `json:"state"`
	Languages   []string `json:"languages"`
	RunnerType  *string  `json:"runner_type"`
	RunnerLabel *string  `json:"runner_label"`
	QuerySuite  string   `json:"query_suite"`
	ThreatModel string   `json:"threat_model"`
	Schedule    *string  `json:"schedule"`
	UpdatedAt   *string  `json:"updated_at"`
}

type DesiredConfig struct {
	State       string   `json:"state"`
	Languages   []string `json:"languages"`
	RunnerType  string   `json:"runner_type"`
	RunnerLabel *string  `json:"runner_label"`
	QuerySuite  string   `json:"query_suite"`
	ThreatModel string   `json:"threat_model"`
}

type Output struct {
	Owner   string         `json:"owner"`
	Repo    string         `json:"repo"`
	Changed bool           `json:"changed"`
	Before  CurrentConfig  `json:"before"`
	After   DesiredConfig  `json:"after"`
	RunID   *int64         `json:"run_id"`
	RunURL  *string        `json:"run_url"`
}

type patchRequest struct {
	State       string   `json:"state"`
	Languages   []string `json:"languages"`
	RunnerType  string   `json:"runner_type"`
	QuerySuite  string   `json:"query_suite"`
	ThreatModel string   `json:"threat_model"`
}

type patchResponse struct {
	RunID  *int64  `json:"run_id"`
	RunURL *string `json:"run_url"`
}
```

- [ ] **Step 4: Add validation and normalization implementation**

Create `internal/codeqldefaultsetup/codeqldefaultsetup.go` with this initial implementation:

```go
package codeqldefaultsetup

import (
	"sort"
	"strings"

	"github.com/y-writings/gh-usecase/internal/validation"
)

var allowedLanguages = map[string]struct{}{
	"actions":                {},
	"c-cpp":                  {},
	"csharp":                 {},
	"go":                     {},
	"java-kotlin":            {},
	"javascript-typescript":  {},
	"python":                 {},
	"ruby":                   {},
	"swift":                  {},
}

var allowedLanguageList = []string{
	"actions",
	"c-cpp",
	"csharp",
	"go",
	"java-kotlin",
	"javascript-typescript",
	"python",
	"ruby",
	"swift",
}

func Validate(input Input) ([]string, error) {
	if input.Owner == "" {
		return nil, validation.New("owner is required")
	}
	if input.Repo == "" {
		return nil, validation.New("repo is required")
	}
	if strings.Contains(input.Repo, "/") {
		return nil, validation.New("repo must not contain /")
	}
	if input.Languages == "" {
		return nil, validation.New("languages is required")
	}

	languages, err := NormalizeLanguages(input.Languages)
	if err != nil {
		return nil, err
	}
	return languages, nil
}

func NormalizeLanguages(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	languages := make(map[string]struct{})

	for _, part := range parts {
		language := strings.TrimSpace(part)
		if language == "" {
			return nil, validation.New("languages must not contain empty values")
		}
		if _, ok := allowedLanguages[language]; !ok {
			return nil, validation.New("languages must contain only: " + strings.Join(allowedLanguageList, ", "))
		}
		languages[language] = struct{}{}
	}

	normalized := make([]string, 0, len(languages))
	for language := range languages {
		normalized = append(normalized, language)
	}
	sort.Strings(normalized)
	return normalized, nil
}
```

- [ ] **Step 5: Run validation tests to verify they pass**

Run:

```bash
go test ./internal/codeqldefaultsetup -run 'TestValidate|TestNormalize' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit Task 3**

```bash
git add internal/codeqldefaultsetup/types.go internal/codeqldefaultsetup/codeqldefaultsetup.go internal/codeqldefaultsetup/codeqldefaultsetup_test.go
git commit -m "feat: validate codeql default setup input"
```

---

## Task 4: Implement Idempotent GET/PATCH Behavior

**Files:**
- Modify: `internal/codeqldefaultsetup/codeqldefaultsetup.go`
- Modify: `internal/codeqldefaultsetup/codeqldefaultsetup_test.go`

- [ ] **Step 1: Add failing execution tests**

Append this test support and tests to `internal/codeqldefaultsetup/codeqldefaultsetup_test.go`:

```go
type restCall struct {
	Method string
	Path   string
	Body   string
}

type fakeRESTClient struct {
	current       CurrentConfig
	patchResponse patchResponse
	errOnGet      error
	errOnPatch    error
	calls         []restCall
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
		return assignJSON(response, c.current)
	case http.MethodPatch:
		if c.errOnPatch != nil {
			return c.errOnPatch
		}
		return assignJSON(response, c.patchResponse)
	default:
		return fmt.Errorf("unexpected method %s", method)
	}
}

func assignJSON(target interface{}, value interface{}) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, target)
}

func TestExecuteDoesNotPatchWhenConfigurationMatches(t *testing.T) {
	runnerType := "standard"
	client := &fakeRESTClient{
		current: CurrentConfig{
			State:       "configured",
			Languages:   []string{"javascript-typescript", "go"},
			RunnerType:  &runnerType,
			QuerySuite:  "default",
			ThreatModel: "remote",
		},
	}

	output, err := Execute(context.Background(), client, Input{Owner: "y-writings", Repo: "repo", Languages: "go,javascript-typescript"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if output.Changed {
		t.Fatal("Changed = true, want false")
	}
	if output.RunID != nil || output.RunURL != nil {
		t.Fatalf("RunID/RunURL = %v/%v, want nil/nil", output.RunID, output.RunURL)
	}
	if len(client.calls) != 1 || client.calls[0].Method != http.MethodGet {
		t.Fatalf("calls = %#v, want only GET", client.calls)
	}
	if client.calls[0].Path != "repos/y-writings/repo/code-scanning/default-setup" {
		t.Fatalf("GET path = %q, want default setup path", client.calls[0].Path)
	}
	if !reflect.DeepEqual(output.Before.Languages, []string{"go", "javascript-typescript"}) {
		t.Fatalf("Before.Languages = %#v, want sorted languages", output.Before.Languages)
	}
}

func TestExecutePatchesWhenConfigurationDiffers(t *testing.T) {
	runID := int64(123456)
	runURL := "https://github.com/y-writings/repo/actions/runs/123456"
	client := &fakeRESTClient{
		current: CurrentConfig{
			State:       "not-configured",
			Languages:   nil,
			QuerySuite:  "",
			ThreatModel: "",
		},
		patchResponse: patchResponse{RunID: &runID, RunURL: &runURL},
	}

	output, err := Execute(context.Background(), client, Input{Owner: "y-writings", Repo: "repo", Languages: "go"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if !output.Changed {
		t.Fatal("Changed = false, want true")
	}
	if output.RunID == nil || *output.RunID != runID {
		t.Fatalf("RunID = %v, want %d", output.RunID, runID)
	}
	if output.RunURL == nil || *output.RunURL != runURL {
		t.Fatalf("RunURL = %v, want %q", output.RunURL, runURL)
	}
	if len(client.calls) != 2 || client.calls[1].Method != http.MethodPatch {
		t.Fatalf("calls = %#v, want GET then PATCH", client.calls)
	}

	wantBody := `{"state":"configured","languages":["go"],"runner_type":"standard","query_suite":"default","threat_model":"remote"}`
	if client.calls[1].Body != wantBody {
		t.Fatalf("PATCH body = %s, want %s", client.calls[1].Body, wantBody)
	}
}

func TestExecutePatchesWhenCurrentLanguagesAreSuperset(t *testing.T) {
	runnerType := "standard"
	client := &fakeRESTClient{
		current: CurrentConfig{
			State:       "configured",
			Languages:   []string{"go", "python"},
			RunnerType:  &runnerType,
			QuerySuite:  "default",
			ThreatModel: "remote",
		},
	}

	output, err := Execute(context.Background(), client, Input{Owner: "owner", Repo: "repo", Languages: "go"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if !output.Changed {
		t.Fatal("Changed = false, want true for language superset")
	}
	if len(client.calls) != 2 || client.calls[1].Method != http.MethodPatch {
		t.Fatalf("calls = %#v, want PATCH", client.calls)
	}
}

func TestExecutePropagatesAPIErrors(t *testing.T) {
	client := &fakeRESTClient{errOnGet: errors.New("github failed")}

	_, err := Execute(context.Background(), client, Input{Owner: "owner", Repo: "repo", Languages: "go"})
	if err == nil || err.Error() != "github failed" {
		t.Fatalf("Execute error = %v, want github failed", err)
	}
}
```

Replace the import block in `internal/codeqldefaultsetup/codeqldefaultsetup_test.go` with:

```go
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)
```

- [ ] **Step 2: Run execution tests to verify they fail**

Run:

```bash
go test ./internal/codeqldefaultsetup -run 'TestExecute' -count=1
```

Expected: FAIL because `Execute` is not defined.

- [ ] **Step 3: Implement execution behavior**

Replace `internal/codeqldefaultsetup/codeqldefaultsetup.go` with:

```go
package codeqldefaultsetup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/y-writings/gh-usecase/internal/githubapi"
	"github.com/y-writings/gh-usecase/internal/validation"
)

var allowedLanguages = map[string]struct{}{
	"actions":               {},
	"c-cpp":                 {},
	"csharp":                {},
	"go":                    {},
	"java-kotlin":           {},
	"javascript-typescript": {},
	"python":                {},
	"ruby":                  {},
	"swift":                 {},
}

var allowedLanguageList = []string{
	"actions",
	"c-cpp",
	"csharp",
	"go",
	"java-kotlin",
	"javascript-typescript",
	"python",
	"ruby",
	"swift",
}

func Execute(ctx context.Context, client githubapi.RESTClient, input Input) (Output, error) {
	languages, err := Validate(input)
	if err != nil {
		return Output{}, err
	}

	path := defaultSetupPath(input.Owner, input.Repo)

	var current CurrentConfig
	if err := client.DoWithContext(ctx, http.MethodGet, path, nil, &current); err != nil {
		return Output{}, err
	}
	current.Languages = normalizeLanguageSlice(current.Languages)

	desired := DesiredConfig{
		State:       "configured",
		Languages:   languages,
		RunnerType:  "standard",
		RunnerLabel: nil,
		QuerySuite:  "default",
		ThreatModel: "remote",
	}

	output := Output{
		Owner:   input.Owner,
		Repo:    input.Repo,
		Changed: false,
		Before:  current,
		After:   desired,
	}

	if configMatches(current, desired) {
		return output, nil
	}

	request := patchRequest{
		State:       desired.State,
		Languages:   desired.Languages,
		RunnerType:  desired.RunnerType,
		QuerySuite:  desired.QuerySuite,
		ThreatModel: desired.ThreatModel,
	}
	body, err := json.Marshal(request)
	if err != nil {
		return Output{}, err
	}

	var patched patchResponse
	if err := client.DoWithContext(ctx, http.MethodPatch, path, bytes.NewReader(body), &patched); err != nil {
		return Output{}, err
	}

	output.Changed = true
	output.RunID = patched.RunID
	output.RunURL = patched.RunURL
	return output, nil
}

func Validate(input Input) ([]string, error) {
	if input.Owner == "" {
		return nil, validation.New("owner is required")
	}
	if input.Repo == "" {
		return nil, validation.New("repo is required")
	}
	if strings.Contains(input.Repo, "/") {
		return nil, validation.New("repo must not contain /")
	}
	if input.Languages == "" {
		return nil, validation.New("languages is required")
	}

	languages, err := NormalizeLanguages(input.Languages)
	if err != nil {
		return nil, err
	}
	return languages, nil
}

func NormalizeLanguages(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	languages := make(map[string]struct{})

	for _, part := range parts {
		language := strings.TrimSpace(part)
		if language == "" {
			return nil, validation.New("languages must not contain empty values")
		}
		if _, ok := allowedLanguages[language]; !ok {
			return nil, validation.New("languages must contain only: " + strings.Join(allowedLanguageList, ", "))
		}
		languages[language] = struct{}{}
	}

	normalized := make([]string, 0, len(languages))
	for language := range languages {
		normalized = append(normalized, language)
	}
	sort.Strings(normalized)
	return normalized, nil
}

func defaultSetupPath(owner string, repo string) string {
	return fmt.Sprintf(
		"repos/%s/%s/code-scanning/default-setup",
		url.PathEscape(owner),
		url.PathEscape(repo),
	)
}

func configMatches(current CurrentConfig, desired DesiredConfig) bool {
	return current.State == desired.State &&
		equalStringSlices(current.Languages, desired.Languages) &&
		stringPointerValue(current.RunnerType) == desired.RunnerType &&
		current.QuerySuite == desired.QuerySuite &&
		current.ThreatModel == desired.ThreatModel
}

func normalizeLanguageSlice(languages []string) []string {
	if len(languages) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(languages))
	for _, language := range languages {
		seen[language] = struct{}{}
	}

	normalized := make([]string, 0, len(seen))
	for language := range seen {
		normalized = append(normalized, language)
	}
	sort.Strings(normalized)
	return normalized
}

func equalStringSlices(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
```

- [ ] **Step 4: Run package tests to verify they pass**

Run:

```bash
go test ./internal/codeqldefaultsetup -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit Task 4**

```bash
git add internal/codeqldefaultsetup/codeqldefaultsetup.go internal/codeqldefaultsetup/codeqldefaultsetup_test.go
git commit -m "feat: configure codeql default setup idempotently"
```

---

## Task 5: Wire CLI Command

**Files:**
- Modify: `internal/cli/usage.go`
- Modify: `internal/cli/runner.go`
- Modify: `internal/cli/runner_test.go`

- [ ] **Step 1: Add failing CLI tests**

Replace the import block in `internal/cli/runner_test.go` with:

```go
import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/y-writings/gh-usecase/internal/githubapi"
)
```

Append these fake client helpers and tests to `internal/cli/runner_test.go`:

```go
type errorRESTClient struct {
	called bool
	err    error
}

func (c *errorRESTClient) DoWithContext(ctx context.Context, method string, path string, body io.Reader, response interface{}) error {
	c.called = true
	return c.err
}

type matchingRESTClient struct {
	calls int
}

func (c *matchingRESTClient) DoWithContext(ctx context.Context, method string, path string, body io.Reader, response interface{}) error {
	c.calls++
	if method != http.MethodGet {
		return errors.New("PATCH must not be called for matching setup")
	}

	payload := map[string]interface{}{
		"state":        "configured",
		"languages":    []string{"go"},
		"runner_type":  "standard",
		"runner_label": nil,
		"query_suite":  "default",
		"threat_model": "remote",
		"schedule":     nil,
		"updated_at":   nil,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, response)
}

func TestRunPrintsCodeQLDefaultSetupUsageWhenHelpFollowsCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"codeql-default-setup", "--help"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "Usage: gh-usecase codeql-default-setup") {
		t.Fatalf("stdout = %q, want codeql usage", stdout.String())
	}
	if !strings.Contains(stdout.String(), "runner_type=standard") {
		t.Fatalf("stdout = %q, want fixed runner type", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunRejectsCodeQLDefaultSetupUnknownOptionBeforeCallingClient(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	restClient := &errorRESTClient{err: errors.New("REST client must not be called")}

	code := runWithClientFactories(
		[]string{"codeql-default-setup", "--owner", "octo", "--repo", "repo", "--languages", "go", "--language", "python"},
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
	if !strings.Contains(stderr.String(), "Usage: gh-usecase codeql-default-setup") {
		t.Fatalf("stderr = %q, want codeql usage", stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown option --language") {
		t.Fatalf("stderr = %q, want unknown option", stderr.String())
	}
	if restClient.called {
		t.Fatal("REST client was called for invalid input")
	}
}

func TestRunRejectsCodeQLDefaultSetupRepeatedLanguagesBeforeCallingClient(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	restClient := &errorRESTClient{err: errors.New("REST client must not be called")}

	code := runWithClientFactories(
		[]string{"codeql-default-setup", "--owner", "octo", "--repo", "repo", "--languages", "go", "--languages", "python"},
		&stdout,
		&stderr,
		func() (githubapi.GraphQLClient, error) { return nil, errors.New("GraphQL client must not be created") },
		func() (githubapi.RESTClient, error) { return restClient, nil },
	)

	if code == 0 {
		t.Fatal("Run exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "languages may be specified only once") {
		t.Fatalf("stderr = %q, want repeated languages error", stderr.String())
	}
	if restClient.called {
		t.Fatal("REST client was called for invalid input")
	}
}

func TestRunRejectsCodeQLDefaultSetupRepoFullNameBeforeCallingClient(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	restClient := &errorRESTClient{err: errors.New("REST client must not be called")}

	code := runWithClientFactories(
		[]string{"codeql-default-setup", "--owner", "octo", "--repo", "octo/repo", "--languages", "go"},
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

func TestRunCodeQLDefaultSetupPrintsJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	restClient := &matchingRESTClient{}

	code := runWithClientFactories(
		[]string{"codeql-default-setup", "--owner", "octo", "--repo", "repo", "--languages", "go"},
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
	if restClient.calls != 1 {
		t.Fatalf("REST calls = %d, want 1", restClient.calls)
	}
	if !strings.Contains(stdout.String(), "\"changed\": false") {
		t.Fatalf("stdout = %q, want changed false JSON", stdout.String())
	}
	if !strings.Contains(stdout.String(), "\"runner_type\": \"standard\"") {
		t.Fatalf("stdout = %q, want runner type", stdout.String())
	}
}
```

Update the existing `TestRunPrintsCommandUsageForHelpWithoutCreatingGitHubClient` table to include the new command:

```go
{command: "codeql-default-setup", usage: "Usage: gh-usecase codeql-default-setup"},
```

- [ ] **Step 2: Run CLI tests to verify they fail**

Run:

```bash
go test ./internal/cli -run 'CodeQL|CommandUsage|RootUsage' -count=1
```

Expected: FAIL because `codeql-default-setup`, `runWithClientFactories`, and usage text are not implemented.

- [ ] **Step 3: Add usage text**

Update `internal/cli/usage.go` so the root usage and new usage include:

```go
const RootUsage = `Usage: gh-usecase <command> [options]

Commands:
  pr-count              Fetch pull request total count
  pr-list               Fetch pull request list
  pr-detail             Fetch pull request detail for analysis
  codeql-default-setup  Configure CodeQL default setup for a repository`
```

Append the new command usage:

```go
const CodeQLDefaultSetupUsage = `Usage: gh-usecase codeql-default-setup --owner <owner> --repo <repo> --languages <csv>

Configure CodeQL default setup with runner_type=standard, query_suite=default, and threat_model=remote.

Languages must be a comma-separated list containing only: actions, c-cpp, csharp, go, java-kotlin, javascript-typescript, python, ruby, swift.`
```

- [ ] **Step 4: Wire command dispatch and REST factory**

Modify `internal/cli/runner.go` imports to include:

```go
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/y-writings/gh-usecase/internal/codeqldefaultsetup"
	"github.com/y-writings/gh-usecase/internal/githubapi"
	"github.com/y-writings/gh-usecase/internal/prcount"
	"github.com/y-writings/gh-usecase/internal/prdetail"
	"github.com/y-writings/gh-usecase/internal/prlist"
	"github.com/y-writings/gh-usecase/internal/validation"
)
```

Add the REST factory type and replace the `Run` / `runWithClientFactory` entry helpers with:

```go
type graphQLClientFactory func() (githubapi.GraphQLClient, error)
type restClientFactory func() (githubapi.RESTClient, error)

func Run(argv []string, stdout io.Writer, stderr io.Writer) int {
	return runWithClientFactories(argv, stdout, stderr, githubapi.NewDefaultGraphQLClient, githubapi.NewDefaultRESTClient)
}

func runWithClientFactory(argv []string, stdout io.Writer, stderr io.Writer, newClient graphQLClientFactory) int {
	return runWithClientFactories(argv, stdout, stderr, newClient, githubapi.NewDefaultRESTClient)
}

func runWithClientFactories(argv []string, stdout io.Writer, stderr io.Writer, newGraphQLClient graphQLClientFactory, newRESTClient restClientFactory) int {
	parsed := ParseArgs(argv)

	if len(argv) == 0 || argv[0] == "--help" || argv[0] == "-h" {
		fmt.Fprintln(stdout, RootUsage)
		return 0
	}
	if len(parsed.Positionals) == 0 {
		fmt.Fprintln(stderr, RootUsage)
		return 1
	}

	command := parsed.Positionals[0]
	if !isKnownCommand(command) {
		fmt.Fprintln(stderr, RootUsage)
		fmt.Fprintf(stderr, "unknown command '%s'\n", command)
		return 1
	}

	switch command {
	case "pr-count":
		return runPrCount(parsed, stdout, stderr, newGraphQLClient)
	case "pr-list":
		return runPrList(parsed, stdout, stderr, newGraphQLClient)
	case "pr-detail":
		return runPrDetail(parsed, stdout, stderr, newGraphQLClient)
	case "codeql-default-setup":
		return runCodeQLDefaultSetup(parsed, stdout, stderr, newRESTClient)
	default:
		fmt.Fprintf(stderr, "command '%s' is not implemented yet\n", command)
		return 1
	}
}
```

Add the new command runner after `runPrDetail`:

```go
func runCodeQLDefaultSetup(parsed ParsedArgs, stdout io.Writer, stderr io.Writer, newClient restClientFactory) int {
	if parsed.Help {
		fmt.Fprintln(stdout, CodeQLDefaultSetupUsage)
		return 0
	}

	if err := rejectUnsupportedOptions(parsed, []string{"owner", "repo", "languages"}); err != nil {
		printCommandError(stderr, CodeQLDefaultSetupUsage, err)
		return 1
	}
	if parsed.OptionOccurrences["languages"] > 1 {
		printCommandError(stderr, CodeQLDefaultSetupUsage, validation.New("languages may be specified only once"))
		return 1
	}

	input := codeqldefaultsetup.Input{
		Owner:     parsed.Options["owner"],
		Repo:      parsed.Options["repo"],
		Languages: parsed.Options["languages"],
	}
	if _, err := codeqldefaultsetup.Validate(input); err != nil {
		printCommandError(stderr, CodeQLDefaultSetupUsage, err)
		return 1
	}

	client, err := newClient()
	if err != nil {
		printExecutionError(stderr, err)
		return 1
	}

	output, err := codeqldefaultsetup.Execute(context.Background(), client, input)
	if err != nil {
		printCommandError(stderr, CodeQLDefaultSetupUsage, err)
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

Add this helper near `printCommandError`:

```go
func rejectUnsupportedOptions(parsed ParsedArgs, allowed []string) error {
	allowedOptions := make(map[string]struct{}, len(allowed))
	for _, option := range allowed {
		allowedOptions[option] = struct{}{}
	}

	unknown := make([]string, 0)
	for option := range parsed.OptionOccurrences {
		if _, ok := allowedOptions[option]; !ok {
			unknown = append(unknown, option)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Strings(unknown)
	return validation.New("unknown option --" + unknown[0])
}
```

Update `isKnownCommand`:

```go
func isKnownCommand(command string) bool {
	switch command {
	case "pr-count", "pr-list", "pr-detail", "codeql-default-setup":
		return true
	default:
		return false
	}
}
```

- [ ] **Step 5: Run CLI tests to verify they pass**

Run:

```bash
go test ./internal/cli -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit Task 5**

```bash
git add internal/cli/usage.go internal/cli/runner.go internal/cli/runner_test.go
git commit -m "feat: wire codeql default setup cli"
```

---

## Task 6: Final Formatting And Verification

**Files:**
- Modify any Go files touched by Tasks 1-5 if formatting changes are produced.

- [ ] **Step 1: Run gofmt**

Run:

```bash
gofmt -w internal/cli/args.go internal/cli/args_test.go internal/githubapi/client.go internal/githubapi/client_test.go internal/codeqldefaultsetup/types.go internal/codeqldefaultsetup/codeqldefaultsetup.go internal/codeqldefaultsetup/codeqldefaultsetup_test.go internal/cli/usage.go internal/cli/runner.go internal/cli/runner_test.go
```

Expected: command exits `0`.

- [ ] **Step 2: Run all tests**

Run:

```bash
go test ./...
```

Expected: PASS for all packages.

- [ ] **Step 3: Inspect final diff**

Run:

```bash
git diff --stat
git diff
```

Expected: diff only includes the CodeQL default setup CLI files and approved spec/plan docs.

- [ ] **Step 4: Commit final formatting if needed**

Only run this if Step 1 changed files after Task 5 commit:

```bash
git add internal/cli/args.go internal/cli/args_test.go internal/githubapi/client.go internal/githubapi/client_test.go internal/codeqldefaultsetup/types.go internal/codeqldefaultsetup/codeqldefaultsetup.go internal/codeqldefaultsetup/codeqldefaultsetup_test.go internal/cli/usage.go internal/cli/runner.go internal/cli/runner_test.go
git commit -m "chore: format codeql default setup cli"
```

---

## Plan Self-Review Checklist

- Spec coverage: command name, required args, language validation, REST client, headers, idempotent GET/PATCH, fixed desired config, JSON output, no waiting, API error behavior, and tests are covered by Tasks 1-6.
- No unsupported scope: no workflow dispatch, no branch protection, no `gh-terraform` inspection, no PR-level rerun logic, no token flags, no dry run.
- Type consistency: `githubapi.RESTClient`, `codeqldefaultsetup.Input`, `CurrentConfig`, `DesiredConfig`, `Output`, `patchRequest`, and `patchResponse` are defined before use.
- Validation clarity: `repo` is non-empty and must not contain `/`; other repository naming rules remain GitHub API responsibility.
