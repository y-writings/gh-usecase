package prcount

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"reflect"
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

	fixture, err := os.ReadFile("../../testdata/pr-count/basic.graphql.json")
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
		"state": []string{"OPEN"},
	}
	if !reflect.DeepEqual(client.variables, wantVariables) {
		t.Fatalf("variables = %#v, want %#v", client.variables, wantVariables)
	}

	got, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent returned error: %v", err)
	}
	got = append(got, '\n')
	want, err := os.ReadFile("../../testdata/pr-count/basic.expected.json")
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("output = %s, want %s", got, want)
	}
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

func TestExecuteRejectsInvalidStateBeforeCallingClient(t *testing.T) {
	client := &fakeClient{}
	state := "DRAFT"

	_, err := Execute(context.Background(), client, Input{Owner: "octokit", Name: "rest.js", State: &state})

	if err == nil {
		t.Fatal("Execute returned nil error, want validation error")
	}
	if client.called {
		t.Fatal("client was called for invalid input")
	}
}
