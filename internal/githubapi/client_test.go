package githubapi

import (
	"context"
	"testing"
)

type compileOnlyClient struct{}

func (compileOnlyClient) DoWithContext(ctx context.Context, query string, variables map[string]interface{}, response interface{}) error {
	return nil
}

func TestGraphQLClientBoundaryCompiles(t *testing.T) {
	var _ GraphQLClient = compileOnlyClient{}
	var _ func() (GraphQLClient, error) = NewDefaultGraphQLClient
}
