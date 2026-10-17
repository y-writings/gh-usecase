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
- `owner-pr-count` - aggregate open pull request counts for all repositories owned by an account
- `pr-list` - fetch pull request list
- `pr-detail` - fetch pull request detail for analysis
- `repo-list` - fetch repositories owned by a user or organization
- `codeql-default-setup` - configure CodeQL default setup for a repository
- `pull-request-creation-policy` - configure who can create pull requests for a repository

Examples:

```sh
./gh-usecase pr-count --owner y-writings --name gh-usecase
./gh-usecase owner-pr-count --owner y-writings
./gh-usecase pr-list --owner y-writings --name gh-usecase --state OPEN --first 10
./gh-usecase pr-detail --owner y-writings --name gh-usecase --number 1
./gh-usecase repo-list --owner y-writings --first 30
./gh-usecase codeql-default-setup --owner y-writings --repo gh-usecase --languages go
./gh-usecase pull-request-creation-policy --owner y-writings --repo gh-usecase --policy collaborators_only
```

## Go Package Usage

### Account-wide open pull request counts

The `ownerprcount` package provides the same complete aggregation used by the
`owner-pr-count` CLI command:

```go
package main

import (
	"context"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/y-writings/gh-usecase/ownerprcount"
)

func main() {
	client, err := api.DefaultGraphQLClient()
	if err != nil {
		panic(err)
	}
	result, err := ownerprcount.Execute(context.Background(), client, ownerprcount.Input{
		Owner: "y-writings",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(result.TotalCount)
}
```

The aggregation includes every repository owned by the specified user or
organization that is visible with the client's authentication, including
repositories with zero open pull requests. Draft pull requests are included;
closed and merged pull requests are not. The package follows all repository
pages internally and uses GitHub's per-repository `totalCount`. The result's
`repositories` field is always an array (including `[]` when none are visible),
and `totalCount` is the sum of its entries.

```json
{
  "owner": "example-org",
  "state": "OPEN",
  "repositories": [
    {"nameWithOwner": "example-org/repo-a", "pullRequestCount": 0},
    {"nameWithOwner": "example-org/repo-b", "pullRequestCount": 3}
  ],
  "totalCount": 3
}
```

An unknown account, GraphQL or network failure, canceled context, missing data,
or failure on any page returns an error and no partial result. Because GitHub can
change while pages are fetched, a successful result reflects observations over
the aggregation period and is not guaranteed to be an atomic point-in-time
snapshot. The CLI converts errors to diagnostic output and a nonzero exit code.

### CodeQL default setup

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
