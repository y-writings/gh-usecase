# gh-usecase

Small Go CLI for GitHub repository use cases.

## Requirements

- Go 1.26.3+
- GitHub authentication available to `gh`

## Build

```sh
go build -o gh-usecase ./cmd/gh-usecase
```

## Docker

```sh
docker build -t gh-usecase .
docker run --rm -e GH_TOKEN gh-usecase <command> [options]
```

## Usage

```sh
./gh-usecase <command> [options]
```

Commands:

- `pr-count` - fetch pull request total count
- `pr-list` - fetch pull request list
- `pr-detail` - fetch pull request detail for analysis
- `repo-list` - fetch repositories owned by a user or organization
- `codeql-default-setup` - configure CodeQL default setup for a repository
- `pull-request-creation-policy` - configure who can create pull requests for a repository

Examples:

```sh
./gh-usecase pr-count --owner y-writings --name gh-usecase
./gh-usecase pr-list --owner y-writings --name gh-usecase --state OPEN --first 10
./gh-usecase pr-detail --owner y-writings --name gh-usecase --number 1
./gh-usecase repo-list --owner y-writings --first 30
./gh-usecase codeql-default-setup --owner y-writings --repo gh-usecase --languages go
./gh-usecase pull-request-creation-policy --owner y-writings --repo gh-usecase --policy collaborators_only
```

## Go Package Usage

The CodeQL default setup reconciler is also available as an importable Go package:

```go
package main

import (
	"context"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/y-writings/gh-usecase/codeqldefaultsetup"
)

func main() {
	client, err := api.NewRESTClient(api.ClientOptions{
		Headers: map[string]string{
			"Accept":               "application/vnd.github+json",
			"X-GitHub-Api-Version": "2022-11-28",
		},
	})
	if err != nil {
		panic(err)
	}

	result, err := codeqldefaultsetup.Reconcile(context.Background(), client, codeqldefaultsetup.Input{
		Owner:     "y-writings",
		Repo:      "gh-usecase",
		Languages: []string{"go"},
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(result.Changed)
}
```

The Go API accepts typed language input as `[]string`; CSV parsing is only part of the CLI adapter.

## License

MIT
