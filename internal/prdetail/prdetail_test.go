package prdetail

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
	fixture   []byte
	query     string
	variables map[string]interface{}
	called    bool
}

func (f *fakeClient) DoWithContext(ctx context.Context, query string, variables map[string]interface{}, response interface{}) error {
	f.called = true
	f.query = query
	f.variables = variables

	fixture := f.fixture
	if fixture == nil {
		var err error
		fixture, err = os.ReadFile("../../testdata/pr-detail/basic.graphql.json")
		if err != nil {
			return err
		}
	}

	return json.Unmarshal(fixture, response)
}

func TestExecuteReturnsFixtureOutputAndBuildsVariables(t *testing.T) {
	client := &fakeClient{}

	output, err := Execute(context.Background(), client, Input{Owner: "octokit", Name: "rest.js", Number: 123})

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if client.query != graphQLQuery {
		t.Fatalf("query = %q, want %q", client.query, graphQLQuery)
	}
	wantVariables := map[string]interface{}{
		"owner":      "octokit",
		"name":       "rest.js",
		"number":     123,
		"filesFirst": 40,
	}
	if !reflect.DeepEqual(client.variables, wantVariables) {
		t.Fatalf("variables = %#v, want %#v", client.variables, wantVariables)
	}

	got, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent returned error: %v", err)
	}
	want, err := os.ReadFile("../../testdata/pr-detail/basic.expected.json")
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	got = compactJSON(t, got)
	want = compactJSON(t, want)
	if !bytes.Equal(got, want) {
		t.Fatalf("output = %s, want %s", got, want)
	}
}

func TestExecuteTransformsNullablePullRequest(t *testing.T) {
	client := &fakeClient{fixture: []byte(`{
		"data": {
			"repository": {
				"pullRequest": null
			}
		}
	}`)}

	output, err := Execute(context.Background(), client, Input{Owner: "octokit", Name: "rest.js", Number: 123})

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	got, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	want := compactJSON(t, []byte(`{
		"data": {
			"repository": {
				"pullRequest": null
			}
		}
	}`))
	if !bytes.Equal(compactJSON(t, got), want) {
		t.Fatalf("output = %s, want %s", got, want)
	}
}

func TestExecuteClassifiesGeneratedFileBeforeBinaryFile(t *testing.T) {
	client := &fakeClient{fixture: []byte(`{
		"data": {
			"repository": {
				"pullRequest": {
					"number": 123,
					"title": "Generated asset",
					"bodyText": "",
					"reviewDecision": null,
					"author": null,
					"mergeCommit": null,
					"baseRefOid": "base-oid",
					"headRefOid": "head-oid",
					"additions": 1,
					"deletions": 0,
					"changedFiles": 1,
					"reviews": { "nodes": [] },
					"reviewThreads": { "nodes": [] },
					"files": {
						"totalCount": 1,
						"nodes": [
							{ "path": "dist/logo.png", "additions": 1, "deletions": 0, "changeType": "ADDED" }
						],
						"pageInfo": { "hasNextPage": false, "endCursor": null }
					},
					"commits": { "nodes": [] }
				}
			}
		}
	}`)}

	output, err := Execute(context.Background(), client, Input{Owner: "octokit", Name: "rest.js", Number: 123})

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	pr := output.Data.Repository.PullRequest
	if pr == nil {
		t.Fatal("PullRequest is nil, want transformed pull request")
	}
	if len(pr.CodeDiff.Files) != 0 {
		t.Fatalf("included files = %#v, want empty", pr.CodeDiff.Files)
	}
	want := []ExcludedFile{{Path: "dist/logo.png", Reason: "likely-generated"}}
	if !reflect.DeepEqual(pr.CodeDiff.ExcludedFiles, want) {
		t.Fatalf("excluded files = %#v, want %#v", pr.CodeDiff.ExcludedFiles, want)
	}
}

func TestExecuteRejectsInvalidNumberBeforeCallingClient(t *testing.T) {
	for _, number := range []int{0, -1} {
		t.Run(strconv.Itoa(number), func(t *testing.T) {
			client := &fakeClient{}

			_, err := Execute(context.Background(), client, Input{Owner: "octokit", Name: "rest.js", Number: number})

			if err == nil {
				t.Fatal("Execute returned nil error, want validation error")
			}
			if client.called {
				t.Fatal("client was called for invalid input")
			}
		})
	}
}

func TestExecuteRejectsInvalidFilesFirstBeforeCallingClient(t *testing.T) {
	for _, filesFirst := range []int{0, 101} {
		t.Run(strconv.Itoa(filesFirst), func(t *testing.T) {
			client := &fakeClient{}

			_, err := Execute(context.Background(), client, Input{Owner: "octokit", Name: "rest.js", Number: 123, FilesFirst: &filesFirst})

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
