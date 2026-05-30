package prcount

import (
	"context"

	"github.com/y-writings/gh-usecase/internal/githubapi"
	"github.com/y-writings/gh-usecase/internal/validation"
)

func Execute(ctx context.Context, client githubapi.GraphQLClient, input Input) (Output, error) {
	if err := Validate(input); err != nil {
		return Output{}, err
	}

	variables := map[string]interface{}{
		"owner": input.Owner,
		"name":  input.Name,
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
	return nil
}
