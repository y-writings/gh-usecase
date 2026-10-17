package ownerprcount

import "context"

// Client is the GraphQL client required by Execute. A go-gh GraphQL client
// satisfies this interface.
type Client interface {
	DoWithContext(ctx context.Context, query string, variables map[string]interface{}, response interface{}) error
}

// Input identifies the GitHub account whose repositories should be aggregated.
type Input struct {
	Owner string
}

// RepositoryCount is the open pull request count for one repository.
type RepositoryCount struct {
	NameWithOwner    string `json:"nameWithOwner"`
	PullRequestCount int    `json:"pullRequestCount"`
}

// Output is a complete aggregation across all owned, visible repositories.
type Output struct {
	Owner        string            `json:"owner"`
	State        string            `json:"state"`
	Repositories []RepositoryCount `json:"repositories"`
	TotalCount   int               `json:"totalCount"`
}

type queryData struct {
	RepositoryOwner *repositoryOwner `json:"repositoryOwner"`
}

type repositoryOwner struct {
	Repositories repositories `json:"repositories"`
}

type repositories struct {
	Nodes    []repositoryNode `json:"nodes"`
	PageInfo pageInfo         `json:"pageInfo"`
}

type repositoryNode struct {
	NameWithOwner *string      `json:"nameWithOwner"`
	PullRequests  pullRequests `json:"pullRequests"`
}

type pullRequests struct {
	TotalCount *int `json:"totalCount"`
}

type pageInfo struct {
	HasNextPage bool    `json:"hasNextPage"`
	EndCursor   *string `json:"endCursor"`
}
