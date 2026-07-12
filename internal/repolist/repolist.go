package repolist

import (
	"context"
	"errors"

	"github.com/y-writings/gh-usecase/internal/githubapi"
	"github.com/y-writings/gh-usecase/internal/validation"
)

func Execute(ctx context.Context, client githubapi.GraphQLClient, input Input) (Output, error) {
	if err := Validate(input); err != nil {
		return Output{}, err
	}

	first := 30
	if input.First != nil {
		first = *input.First
	}
	variables := map[string]interface{}{
		"owner": input.Owner,
		"first": first,
	}
	if input.After != nil {
		variables["after"] = *input.After
	}

	var data Data
	if err := client.DoWithContext(ctx, graphQLQuery, variables, &data); err != nil {
		return Output{}, err
	}
	if data.RepositoryOwner == nil {
		return Output{}, errors.New("repository owner not found")
	}
	return Output{Data: data}, nil
}

func Validate(input Input) error {
	if input.Owner == "" {
		return validation.New("owner is required")
	}
	if input.First != nil && (*input.First < 1 || *input.First > 100) {
		return validation.New("first must be between 1 and 100")
	}
	if input.After != nil && *input.After == "" {
		return validation.New("after must not be empty")
	}
	return nil
}
