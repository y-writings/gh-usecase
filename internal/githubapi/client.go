package githubapi

import (
	"context"

	"github.com/cli/go-gh/v2/pkg/api"
)

type GraphQLClient interface {
	DoWithContext(ctx context.Context, query string, variables map[string]interface{}, response interface{}) error
}

func NewDefaultGraphQLClient() (GraphQLClient, error) {
	return api.DefaultGraphQLClient()
}
