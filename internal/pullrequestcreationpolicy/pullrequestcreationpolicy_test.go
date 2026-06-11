package pullrequestcreationpolicy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
)

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
