package codeqldefaultsetup

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
