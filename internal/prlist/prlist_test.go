package prlist

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"testing"
)

type fakeClient struct {
	query     string
	variables map[string]interface{}
	called    bool
}

func (f *fakeClient) DoWithContext(ctx context.Context, query string, variables map[string]interface{}, response interface{}) error {
	f.called = true
	f.query = query
	f.variables = variables

	fixture, err := os.ReadFile("../../testdata/pr-list/basic.graphql.json")
	if err != nil {
		return err
	}

	return json.Unmarshal(fixture, response)
}

func TestExecuteReturnsFixtureOutputAndBuildsVariables(t *testing.T) {
	client := &fakeClient{}
	state := "OPEN"

	output, err := Execute(context.Background(), client, Input{Owner: "octokit", Name: "rest.js", State: &state})

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if client.query != graphQLQuery {
		t.Fatalf("query = %q, want %q", client.query, graphQLQuery)
	}
	wantVariables := map[string]interface{}{
		"owner": "octokit",
		"name":  "rest.js",
		"first": 30,
		"state": []string{"OPEN"},
	}
	if !reflect.DeepEqual(client.variables, wantVariables) {
		t.Fatalf("variables = %#v, want %#v", client.variables, wantVariables)
	}

	got, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent returned error: %v", err)
	}
	want, err := os.ReadFile("../../testdata/pr-list/basic.expected.json")
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	got = compactJSON(t, got)
	want = compactJSON(t, want)
	if !bytes.Equal(got, want) {
		t.Fatalf("output = %s, want %s", got, want)
	}
}

func TestExecuteIncludesAfterWhenProvided(t *testing.T) {
	client := &fakeClient{}
	after := "cursor-1"

	_, err := Execute(context.Background(), client, Input{Owner: "octokit", Name: "rest.js", After: &after})

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if got := client.variables["after"]; got != "cursor-1" {
		t.Fatalf("after variable = %#v, want cursor-1", got)
	}
}

func TestExecuteIncludesAfterWhenExplicitlyEmpty(t *testing.T) {
	client := &fakeClient{}
	after := ""

	_, err := Execute(context.Background(), client, Input{Owner: "octokit", Name: "rest.js", After: &after})

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	afterVariable, ok := client.variables["after"]
	if !ok {
		t.Fatalf("variables omits explicitly empty after: %#v", client.variables)
	}
	if afterVariable != "" {
		t.Fatalf("after variable = %#v, want empty string", afterVariable)
	}
}

func TestExecuteRejectsInvalidFirstBeforeCallingClient(t *testing.T) {
	for _, first := range []int{0, 101} {
		t.Run(strconv.Itoa(first), func(t *testing.T) {
			client := &fakeClient{}

			_, err := Execute(context.Background(), client, Input{Owner: "octokit", Name: "rest.js", First: &first})

			if err == nil {
				t.Fatal("Execute returned nil error, want validation error")
			}
			if client.called {
				t.Fatal("client was called for invalid input")
			}
		})
	}
}

func compactJSON(t *testing.T, input []byte) []byte {
	t.Helper()

	var output bytes.Buffer
	if err := json.Compact(&output, input); err != nil {
		t.Fatalf("Compact returned error: %v", err)
	}
	return output.Bytes()
}

func TestExecuteOmitsEmptyStateVariable(t *testing.T) {
	client := &fakeClient{}

	_, err := Execute(context.Background(), client, Input{Owner: "octokit", Name: "rest.js"})

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if _, ok := client.variables["state"]; ok {
		t.Fatalf("variables includes state for empty input: %#v", client.variables)
	}
}
