package githubapi

import (
	"context"
	"io"
	"testing"
)

type compileOnlyGraphQLClient struct{}

func (compileOnlyGraphQLClient) DoWithContext(ctx context.Context, query string, variables map[string]interface{}, response interface{}) error {
	return nil
}

type compileOnlyRESTClient struct{}

func (compileOnlyRESTClient) DoWithContext(ctx context.Context, method string, path string, body io.Reader, response interface{}) error {
	return nil
}

func TestGraphQLClientBoundaryCompiles(t *testing.T) {
	var _ GraphQLClient = compileOnlyGraphQLClient{}
	var _ func() (GraphQLClient, error) = NewDefaultGraphQLClient
}

func TestRESTClientBoundaryCompiles(t *testing.T) {
	var _ RESTClient = compileOnlyRESTClient{}
	var _ func() (RESTClient, error) = NewDefaultRESTClient
}
