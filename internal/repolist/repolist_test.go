package repolist

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type fakeGraphQLClient struct {
	fixture   []byte
	err       error
	query     string
	variables map[string]interface{}
	calls     int
}

func (f *fakeGraphQLClient) DoWithContext(ctx context.Context, query string, variables map[string]interface{}, response interface{}) error {
	f.calls++
	f.query = query
	f.variables = variables
	if f.err != nil {
		return f.err
	}
	return json.Unmarshal(f.fixture, response)
}

func TestGraphQLQueryShape(t *testing.T) {
	want := `query ($owner: String!, $first: Int!, $after: String) {
  repositoryOwner(login: $owner) {
    repositories(first: $first, after: $after, ownerAffiliations: OWNER, orderBy: {field: NAME, direction: ASC}) {
      nodes {
        name
        nameWithOwner
        url
      }
      pageInfo {
        hasNextPage
        endCursor
      }
    }
  }
}`

	if graphQLQuery != want {
		t.Fatalf("graphQLQuery = %q, want %q", graphQLQuery, want)
	}
	if strings.Count(graphQLQuery, "repositories(") != 1 {
		t.Fatalf("graphQLQuery repositories connection count = %d, want 1", strings.Count(graphQLQuery, "repositories("))
	}
	for _, forbidden := range []string{"isFork", "privacy", "isArchived"} {
		if strings.Contains(graphQLQuery, forbidden) {
			t.Fatalf("graphQLQuery contains forbidden filter %q", forbidden)
		}
	}
}

func TestExecuteUsesDefaultPaginationAndReturnsRepositories(t *testing.T) {
	client := &fakeGraphQLClient{fixture: []byte(`{
		"repositoryOwner": {
			"repositories": {
				"nodes": [
					{"name": "alpha", "nameWithOwner": "octo/alpha", "url": "https://github.com/octo/alpha"},
					{"name": "beta", "nameWithOwner": "octo/beta", "url": "https://github.com/octo/beta"}
				],
				"pageInfo": {"hasNextPage": true, "endCursor": "cursor-2"}
			}
		}
	}`)}

	output, err := Execute(context.Background(), client, Input{Owner: "octo"})

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if client.calls != 1 {
		t.Fatalf("client calls = %d, want 1", client.calls)
	}
	if client.query != graphQLQuery {
		t.Fatalf("query = %q, want graphQLQuery", client.query)
	}
	wantVariables := map[string]interface{}{"owner": "octo", "first": 30}
	if !reflect.DeepEqual(client.variables, wantVariables) {
		t.Fatalf("variables = %#v, want %#v", client.variables, wantVariables)
	}
	endCursor := "cursor-2"
	wantOutput := Output{Data: Data{RepositoryOwner: &RepositoryOwner{Repositories: Repositories{
		Nodes: []RepositoryNode{
			{Name: "alpha", NameWithOwner: "octo/alpha", URL: "https://github.com/octo/alpha"},
			{Name: "beta", NameWithOwner: "octo/beta", URL: "https://github.com/octo/beta"},
		},
		PageInfo: PageInfo{HasNextPage: true, EndCursor: &endCursor},
	}}}}
	if !reflect.DeepEqual(output, wantOutput) {
		t.Fatalf("output = %#v, want %#v", output, wantOutput)
	}
}

func TestExecutePassesThroughExplicitPagination(t *testing.T) {
	client := &fakeGraphQLClient{fixture: []byte(`{
		"repositoryOwner": {
			"repositories": {
				"nodes": [],
				"pageInfo": {"hasNextPage": false, "endCursor": null}
			}
		}
	}`)}
	first := 100
	after := "cursor-1"

	_, err := Execute(context.Background(), client, Input{Owner: "octo", First: &first, After: &after})

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	wantVariables := map[string]interface{}{"owner": "octo", "first": 100, "after": "cursor-1"}
	if !reflect.DeepEqual(client.variables, wantVariables) {
		t.Fatalf("variables = %#v, want %#v", client.variables, wantVariables)
	}
}

func TestExecutePreservesEmptyNodesAndNullEndCursor(t *testing.T) {
	client := &fakeGraphQLClient{fixture: []byte(`{
		"repositoryOwner": {
			"repositories": {
				"nodes": [],
				"pageInfo": {"hasNextPage": false, "endCursor": null}
			}
		}
	}`)}

	output, err := Execute(context.Background(), client, Input{Owner: "octo"})

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	repositories := output.Data.RepositoryOwner.Repositories
	if repositories.Nodes == nil {
		t.Fatal("nodes = nil, want non-nil empty slice")
	}
	if len(repositories.Nodes) != 0 {
		t.Fatalf("nodes = %#v, want empty slice", repositories.Nodes)
	}
	if repositories.PageInfo.EndCursor != nil {
		t.Fatalf("endCursor = %#v, want nil", repositories.PageInfo.EndCursor)
	}
}

func TestExecuteRejectsInvalidInputBeforeCallingClient(t *testing.T) {
	client := &fakeGraphQLClient{}

	_, err := Execute(context.Background(), client, Input{})

	if err == nil {
		t.Fatal("Execute returned nil error, want validation error")
	}
	if client.calls != 0 {
		t.Fatalf("client calls = %d, want 0", client.calls)
	}
}

func TestExecuteRejectsUnknownRepositoryOwner(t *testing.T) {
	client := &fakeGraphQLClient{fixture: []byte(`{"repositoryOwner":null}`)}

	_, err := Execute(context.Background(), client, Input{Owner: "missing"})

	if err == nil {
		t.Fatal("Execute returned nil error, want repository owner error")
	}
	if err.Error() != "repository owner not found" {
		t.Fatalf("Execute error = %q, want %q", err, "repository owner not found")
	}
}

func TestExecutePropagatesGraphQLErrorWithoutRetry(t *testing.T) {
	wantErr := errors.New("graphql failed")
	client := &fakeGraphQLClient{err: wantErr}

	_, err := Execute(context.Background(), client, Input{Owner: "octo"})

	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute error = %v, want %v", err, wantErr)
	}
	if client.calls != 1 {
		t.Fatalf("client calls = %d, want 1", client.calls)
	}
}

func TestValidateRejectsInvalidInput(t *testing.T) {
	zero := 0
	overMaximum := 101
	empty := ""
	tests := []struct {
		name  string
		input Input
		want  string
	}{
		{name: "missing owner", input: Input{}, want: "owner is required"},
		{name: "first below minimum", input: Input{Owner: "octokit", First: &zero}, want: "first must be between 1 and 100"},
		{name: "first above maximum", input: Input{Owner: "octokit", First: &overMaximum}, want: "first must be between 1 and 100"},
		{name: "explicitly empty after", input: Input{Owner: "octokit", After: &empty}, want: "after must not be empty"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := Validate(test.input)
			if err == nil {
				t.Fatal("Validate returned nil error, want validation error")
			}
			if err.Error() != test.want {
				t.Fatalf("Validate error = %q, want %q", err, test.want)
			}
		})
	}
}

func TestValidateAcceptsValidInput(t *testing.T) {
	one := 1
	hundred := 100
	tests := []struct {
		name  string
		input Input
	}{
		{name: "minimum first", input: Input{Owner: "octokit", First: &one}},
		{name: "maximum first", input: Input{Owner: "octokit", First: &hundred}},
		{name: "omitted pagination", input: Input{Owner: "octokit"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := Validate(test.input); err != nil {
				t.Fatalf("Validate returned error: %v", err)
			}
		})
	}
}
