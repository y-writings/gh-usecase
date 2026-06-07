package githubapi

import (
	"context"
	"io"

	"github.com/cli/go-gh/v2/pkg/api"
)

const restAPIVersion = "2022-11-28"

type GraphQLClient interface {
	DoWithContext(ctx context.Context, query string, variables map[string]interface{}, response interface{}) error
}

type RESTClient interface {
	DoWithContext(ctx context.Context, method string, path string, body io.Reader, response interface{}) error
}

func NewDefaultGraphQLClient() (GraphQLClient, error) {
	return api.DefaultGraphQLClient()
}

func NewDefaultRESTClient() (RESTClient, error) {
	return api.NewRESTClient(api.ClientOptions{
		Headers: map[string]string{
			"Accept":               "application/vnd.github+json",
			"X-GitHub-Api-Version": restAPIVersion,
		},
	})
}
