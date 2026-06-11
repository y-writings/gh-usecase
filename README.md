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
- `codeql-default-setup` - configure CodeQL default setup for a repository

Examples:

```sh
./gh-usecase pr-count --owner y-writings --name gh-usecase
./gh-usecase pr-list --owner y-writings --name gh-usecase --state OPEN --first 10
./gh-usecase pr-detail --owner y-writings --name gh-usecase --number 1
./gh-usecase codeql-default-setup --owner y-writings --repo gh-usecase --languages go
```

## License

MIT
