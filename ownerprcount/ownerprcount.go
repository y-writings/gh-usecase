package ownerprcount

import (
	"context"
	"errors"
	"fmt"

	"github.com/y-writings/gh-usecase/internal/validation"
)

const repositoryPageSize = 100

// Execute fetches and aggregates every page of repositories owned by input.Owner.
// It never returns a partial Output when any page cannot be fetched or decoded.
func Execute(ctx context.Context, client Client, input Input) (Output, error) {
	if err := Validate(input); err != nil {
		return Output{}, err
	}

	output := Output{
		Owner:        input.Owner,
		State:        "OPEN",
		Repositories: make([]RepositoryCount, 0),
	}
	var after *string

	for {
		if err := ctx.Err(); err != nil {
			return Output{}, err
		}

		variables := map[string]interface{}{
			"owner": input.Owner,
			"first": repositoryPageSize,
			"after": after,
		}
		var data queryData
		if err := client.DoWithContext(ctx, graphQLQuery, variables, &data); err != nil {
			return Output{}, err
		}
		if data.RepositoryOwner == nil {
			return Output{}, errors.New("repository owner not found")
		}

		page := data.RepositoryOwner.Repositories
		for _, node := range page.Nodes {
			if node.NameWithOwner == nil || *node.NameWithOwner == "" {
				return Output{}, errors.New("repository nameWithOwner is missing")
			}
			if node.PullRequests.TotalCount == nil {
				return Output{}, fmt.Errorf("open pull request totalCount is missing for %s", *node.NameWithOwner)
			}
			if *node.PullRequests.TotalCount < 0 {
				return Output{}, fmt.Errorf("open pull request totalCount is negative for %s", *node.NameWithOwner)
			}
			output.Repositories = append(output.Repositories, RepositoryCount{
				NameWithOwner:    *node.NameWithOwner,
				PullRequestCount: *node.PullRequests.TotalCount,
			})
			output.TotalCount += *node.PullRequests.TotalCount
		}

		if !page.PageInfo.HasNextPage {
			return output, nil
		}
		if page.PageInfo.EndCursor == nil || *page.PageInfo.EndCursor == "" {
			return Output{}, errors.New("repository page hasNextPage is true but endCursor is missing")
		}
		if after != nil && *page.PageInfo.EndCursor == *after {
			return Output{}, errors.New("repository pagination cursor did not advance")
		}
		after = page.PageInfo.EndCursor
	}
}

// Validate checks input without issuing a GitHub request.
func Validate(input Input) error {
	if input.Owner == "" {
		return validation.New("owner is required")
	}
	return nil
}
