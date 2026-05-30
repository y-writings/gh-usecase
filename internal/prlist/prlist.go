package prlist

import (
	"context"

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
		"name":  input.Name,
		"first": first,
	}
	if input.After != nil {
		variables["after"] = *input.After
	}
	if input.State != nil {
		variables["state"] = []string{*input.State}
	}

	var output Output
	if err := client.DoWithContext(ctx, graphQLQuery, variables, &output); err != nil {
		return Output{}, err
	}

	return output, nil
}

func Validate(input Input) error {
	if input.Owner == "" {
		return validation.New("owner is required")
	}
	if input.Name == "" {
		return validation.New("name is required")
	}
	if input.State != nil && *input.State != "OPEN" && *input.State != "CLOSED" && *input.State != "MERGED" {
		return validation.New("state must be OPEN, CLOSED, or MERGED")
	}

	first := 30
	if input.First != nil {
		first = *input.First
	}
	if first < 1 || first > 100 {
		return validation.New("first must be between 1 and 100")
	}
	return nil
}
