package ownerprcount

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type response struct {
	payload string
	err     error
}

type fakeClient struct {
	responses []response
	queries   []string
	variables []map[string]interface{}
}

func (f *fakeClient) DoWithContext(_ context.Context, query string, variables map[string]interface{}, target interface{}) error {
	f.queries = append(f.queries, query)
	f.variables = append(f.variables, variables)
	item := f.responses[len(f.queries)-1]
	if item.err != nil {
		return item.err
	}
	return json.Unmarshal([]byte(item.payload), target)
}

func TestExecuteAggregatesEveryRepositoryPage(t *testing.T) {
	client := &fakeClient{responses: []response{
		{payload: `{"repositoryOwner":{"repositories":{"nodes":[{"nameWithOwner":"octo/alpha","pullRequests":{"totalCount":0}},{"nameWithOwner":"octo/beta","pullRequests":{"totalCount":3}}],"pageInfo":{"hasNextPage":true,"endCursor":"next"}}}}`},
		{payload: `{"repositoryOwner":{"repositories":{"nodes":[{"nameWithOwner":"octo/gamma","pullRequests":{"totalCount":2}}],"pageInfo":{"hasNextPage":false,"endCursor":null}}}}`},
	}}

	got, err := Execute(context.Background(), client, Input{Owner: "octo"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	want := Output{Owner: "octo", State: "OPEN", Repositories: []RepositoryCount{
		{NameWithOwner: "octo/alpha", PullRequestCount: 0},
		{NameWithOwner: "octo/beta", PullRequestCount: 3},
		{NameWithOwner: "octo/gamma", PullRequestCount: 2},
	}, TotalCount: 5}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Execute output = %#v, want %#v", got, want)
	}
	if len(client.queries) != 2 || client.queries[0] != graphQLQuery {
		t.Fatalf("queries = %#v, want two aggregation queries", client.queries)
	}
	if client.variables[0]["owner"] != "octo" || client.variables[0]["first"] != 100 || client.variables[0]["after"] != (*string)(nil) {
		t.Fatalf("first-page variables = %#v", client.variables[0])
	}
	after, ok := client.variables[1]["after"].(*string)
	if !ok || after == nil || *after != "next" {
		t.Fatalf("second-page after = %#v, want next", client.variables[1]["after"])
	}
}

func TestExecuteReturnsNonNilEmptyRepositories(t *testing.T) {
	client := &fakeClient{responses: []response{{payload: `{"repositoryOwner":{"repositories":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":null}}}}`}}}

	got, err := Execute(context.Background(), client, Input{Owner: "empty"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if got.Repositories == nil || len(got.Repositories) != 0 || got.TotalCount != 0 {
		t.Fatalf("Execute output = %#v, want non-nil empty repositories and zero total", got)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"repositories":[]`) {
		t.Fatalf("JSON = %s, want repositories array", encoded)
	}
}

func TestExecuteFailsWithoutReturningPartialAggregation(t *testing.T) {
	wantErr := errors.New("second page failed")
	client := &fakeClient{responses: []response{
		{payload: `{"repositoryOwner":{"repositories":{"nodes":[{"nameWithOwner":"octo/alpha","pullRequests":{"totalCount":4}}],"pageInfo":{"hasNextPage":true,"endCursor":"next"}}}}`},
		{err: wantErr},
	}}

	got, err := Execute(context.Background(), client, Input{Owner: "octo"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute error = %v, want %v", err, wantErr)
	}
	if !reflect.DeepEqual(got, Output{}) {
		t.Fatalf("Execute output = %#v, want zero output", got)
	}
}

func TestExecuteRejectsMissingOwnerAndData(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    string
	}{
		{name: "unknown owner", payload: `{"repositoryOwner":null}`, want: "repository owner not found"},
		{name: "missing name", payload: `{"repositoryOwner":{"repositories":{"nodes":[{"pullRequests":{"totalCount":1}}],"pageInfo":{}}}}`, want: "repository nameWithOwner is missing"},
		{name: "missing count", payload: `{"repositoryOwner":{"repositories":{"nodes":[{"nameWithOwner":"octo/repo","pullRequests":{}}],"pageInfo":{}}}}`, want: "totalCount is missing"},
		{name: "missing cursor", payload: `{"repositoryOwner":{"repositories":{"nodes":[],"pageInfo":{"hasNextPage":true,"endCursor":null}}}}`, want: "endCursor is missing"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &fakeClient{responses: []response{{payload: test.payload}}}
			_, err := Execute(context.Background(), client, Input{Owner: "octo"})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Execute error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestExecuteValidatesAndHonorsCanceledContext(t *testing.T) {
	client := &fakeClient{}
	if _, err := Execute(context.Background(), client, Input{}); err == nil || err.Error() != "owner is required" {
		t.Fatalf("missing owner error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Execute(ctx, client, Input{Owner: "octo"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled error = %v, want context.Canceled", err)
	}
	if len(client.queries) != 0 {
		t.Fatalf("client calls = %d, want zero", len(client.queries))
	}
}

func TestQueryRequestsOwnedRepositoriesAndOpenTotalCount(t *testing.T) {
	for _, fragment := range []string{"repositoryOwner(login: $owner)", "ownerAffiliations: OWNER", "pullRequests(states: [OPEN])", "totalCount", "hasNextPage", "endCursor"} {
		if !strings.Contains(graphQLQuery, fragment) {
			t.Fatalf("query missing %q", fragment)
		}
	}
}
