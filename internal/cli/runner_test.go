package cli

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

type errorGraphQLClient struct {
	called bool
	err    error
}

func (c *errorGraphQLClient) DoWithContext(ctx context.Context, query string, variables map[string]interface{}, response interface{}) error {
	c.called = true
	return c.err
}

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

func TestRunPrintsRootUsageForNoArgs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run(nil, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "Usage: gh-usecase") {
		t.Fatalf("stdout = %q, want root usage", stdout.String())
	}
	if strings.Contains(stdout.String(), "bun run") {
		t.Fatalf("stdout = %q, must not mention bun run", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunPrintsRootUsageForHelp(t *testing.T) {
	for _, flag := range []string{"--help", "-h"} {
		t.Run(flag, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run([]string{flag}, &stdout, &stderr)

			if code != 0 {
				t.Fatalf("Run exit code = %d, want 0", code)
			}
			if !strings.Contains(stdout.String(), "Usage: gh-usecase") {
				t.Fatalf("stdout = %q, want root usage", stdout.String())
			}
			if stderr.String() != "" {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
		})
	}
}

func TestRunReportsUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"wat"}, &stdout, &stderr)

	if code == 0 {
		t.Fatal("Run exit code = 0, want non-zero")
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Usage: gh-usecase") {
		t.Fatalf("stderr = %q, want root usage", stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown command 'wat'") {
		t.Fatalf("stderr = %q, want unknown command message", stderr.String())
	}
}

func TestRunReportsUnknownCommandWhenHelpFollowsUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"wat", "--help"}, &stdout, &stderr)

	if code == 0 {
		t.Fatal("Run exit code = 0, want non-zero")
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Usage: gh-usecase") {
		t.Fatalf("stderr = %q, want root usage", stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown command 'wat'") {
		t.Fatalf("stderr = %q, want unknown command message", stderr.String())
	}
}

func TestRunReportsUsageForOptionOnlyArgs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"--owner", "octo"}, &stdout, &stderr)

	if code == 0 {
		t.Fatal("Run exit code = 0, want non-zero")
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Usage: gh-usecase") {
		t.Fatalf("stderr = %q, want root usage", stderr.String())
	}
}

func TestRunReportsValidationFailureWithCommandUsage(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"pr-detail", "--number", "not-a-number"}, &stdout, &stderr)

	if code == 0 {
		t.Fatal("Run exit code = 0, want non-zero")
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Usage: gh-usecase pr-detail") {
		t.Fatalf("stderr = %q, want pr-detail usage", stderr.String())
	}
	if !strings.Contains(stderr.String(), "number must be an integer") {
		t.Fatalf("stderr = %q, want validation message", stderr.String())
	}
}

func TestRunRejectsEmptyPrCountStateWithUsageBeforeCallingClient(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	client := &errorGraphQLClient{err: errors.New("client must not be called")}

	code := runWithClientFactory([]string{"pr-count", "--owner", "octokit", "--name", "rest.js", "--state="}, &stdout, &stderr, func() (githubapi.GraphQLClient, error) {
		return client, nil
	})

	if code == 0 {
		t.Fatal("Run exit code = 0, want non-zero")
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Usage: gh-usecase pr-count") {
		t.Fatalf("stderr = %q, want pr-count usage", stderr.String())
	}
	if !strings.Contains(stderr.String(), "state must be OPEN, CLOSED, or MERGED") {
		t.Fatalf("stderr = %q, want validation message", stderr.String())
	}
	if client.called {
		t.Fatal("GitHub client was called for invalid input")
	}
}

func TestRunRejectsEmptyPrListFirstWithUsageBeforeCallingClient(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	client := &errorGraphQLClient{err: errors.New("client must not be called")}

	code := runWithClientFactory([]string{"pr-list", "--owner", "octokit", "--name", "rest.js", "--first="}, &stdout, &stderr, func() (githubapi.GraphQLClient, error) {
		return client, nil
	})

	if code == 0 {
		t.Fatal("Run exit code = 0, want non-zero")
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Usage: gh-usecase pr-list") {
		t.Fatalf("stderr = %q, want pr-list usage", stderr.String())
	}
	if !strings.Contains(stderr.String(), "first must be an integer") {
		t.Fatalf("stderr = %q, want integer validation message", stderr.String())
	}
	if client.called {
		t.Fatal("GitHub client was called for invalid input")
	}
}

func TestRunRejectsEmptyPrDetailFilesFirstWithUsageBeforeCallingClient(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	client := &errorGraphQLClient{err: errors.New("client must not be called")}

	code := runWithClientFactory([]string{"pr-detail", "--owner", "octokit", "--name", "rest.js", "--number", "1", "--filesFirst="}, &stdout, &stderr, func() (githubapi.GraphQLClient, error) {
		return client, nil
	})

	if code == 0 {
		t.Fatal("Run exit code = 0, want non-zero")
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Usage: gh-usecase pr-detail") {
		t.Fatalf("stderr = %q, want pr-detail usage", stderr.String())
	}
	if !strings.Contains(stderr.String(), "filesFirst must be an integer") {
		t.Fatalf("stderr = %q, want integer validation message", stderr.String())
	}
	if client.called {
		t.Fatal("GitHub client was called for invalid input")
	}
}

func TestRunPrintsExecutionErrorWithCommandFailureContext(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	client := &errorGraphQLClient{err: errors.New("graphql exploded")}

	code := runWithClientFactory([]string{"pr-count", "--owner", "octokit", "--name", "rest.js"}, &stdout, &stderr, func() (githubapi.GraphQLClient, error) {
		return client, nil
	})

	if code == 0 {
		t.Fatal("Run exit code = 0, want non-zero")
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if strings.Contains(stderr.String(), "Usage: gh-usecase pr-count") {
		t.Fatalf("stderr = %q, must not include validation usage", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Failed to execute command: graphql exploded") {
		t.Fatalf("stderr = %q, want command failure context", stderr.String())
	}
	if !client.called {
		t.Fatal("GitHub client was not called for valid input")
	}
}

func TestRunPrintsPrCountUsageWhenHelpFollowsCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"pr-count", "--help"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "Usage: gh-usecase pr-count") {
		t.Fatalf("stdout = %q, want pr-count usage", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunPrintsCommandUsageForHelpWithoutCreatingGitHubClient(t *testing.T) {
	for _, tc := range []struct {
		command string
		usage   string
	}{
		{command: "pr-count", usage: "Usage: gh-usecase pr-count"},
		{command: "pr-list", usage: "Usage: gh-usecase pr-list"},
		{command: "pr-detail", usage: "Usage: gh-usecase pr-detail"},
		{command: "codeql-default-setup", usage: "Usage: gh-usecase codeql-default-setup"},
	} {
		t.Run(tc.command, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			clientCalls := 0

			code := runWithClientFactory([]string{tc.command, "--help"}, &stdout, &stderr, func() (githubapi.GraphQLClient, error) {
				clientCalls++
				return nil, errors.New("GitHub client must not be created for help")
			})

			if code != 0 {
				t.Fatalf("Run exit code = %d, want 0", code)
			}
			if !strings.Contains(stdout.String(), tc.usage) {
				t.Fatalf("stdout = %q, want command usage", stdout.String())
			}
			if stderr.String() != "" {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
			if clientCalls != 0 {
				t.Fatalf("GitHub client calls = %d, want 0", clientCalls)
			}
		})
	}
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

func TestRunPrintsPrListUsageWhenHelpFollowsCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"pr-list", "--help"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "Usage: gh-usecase pr-list") {
		t.Fatalf("stdout = %q, want pr-list usage", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunPrintsPrDetailUsageWhenHelpFollowsCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"pr-detail", "--help"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "Usage: gh-usecase pr-detail") {
		t.Fatalf("stdout = %q, want pr-detail usage", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}
