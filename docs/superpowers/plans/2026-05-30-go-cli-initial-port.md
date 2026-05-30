# Go CLI Initial Port Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the initial Go CLI port of `pr-count`, `pr-list`, and `pr-detail` while preserving current Bun behavior through shared fixture/golden tests.

**Architecture:** Keep Bun as-is for npm entrypoints, but extract testable transformation boundaries. Add `cmd/gh-usecase` plus focused `internal/*` Go packages. Use `go-gh/v2` through a small `GraphQLClient` interface and offline fixture tests for parity.

**Tech Stack:** Bun, TypeScript, Zod, Go 1.26.x, `github.com/cli/go-gh/v2/pkg/api`, JSON fixture tests.

---

## Files To Create Or Modify

- Create: `testdata/pr-count/basic.graphql.json` and `testdata/pr-count/basic.expected.json` for count parity.
- Create: `testdata/pr-list/basic.graphql.json` and `testdata/pr-list/basic.expected.json` for list parity.
- Create: `testdata/pr-detail/basic.graphql.json` and `testdata/pr-detail/basic.expected.json` for detail transform parity.
- Create: `src/cli/pr-count/query.ts`, `src/cli/pr-list/query.ts`, `src/cli/pr-detail/transform.ts` to expose pure Bun behavior for tests.
- Modify: `src/cli/pr-count/index.ts`, `src/cli/pr-list/index.ts`, `src/cli/pr-detail/index.ts` to call extracted functions.
- Create: `src/cli/**/fixture.test.ts` files for Bun fixture tests.
- Modify: `.mise/config.toml` to pin Go 1.26.x.
- Create: `go.mod` and `go.sum`.
- Create: `cmd/gh-usecase/main.go` for Go CLI entrypoint.
- Create: `internal/cli/args.go`, `internal/cli/runner.go`, `internal/cli/usage.go` for dispatch, parser, usage, and JSON output.
- Create: `internal/githubapi/client.go` for the `GraphQLClient` interface and production client construction.
- Create: `internal/prcount/prcount.go`, `internal/prcount/query.go`, `internal/prcount/types.go`, `internal/prcount/prcount_test.go`.
- Create: `internal/prlist/prlist.go`, `internal/prlist/query.go`, `internal/prlist/types.go`, `internal/prlist/prlist_test.go`.
- Create: `internal/prdetail/prdetail.go`, `internal/prdetail/query.go`, `internal/prdetail/types.go`, `internal/prdetail/file_utils.go`, `internal/prdetail/prdetail_test.go`.

## Task 1: Add Shared JSON Fixtures

**Files:**
- Create: `testdata/pr-count/basic.graphql.json`
- Create: `testdata/pr-count/basic.expected.json`
- Create: `testdata/pr-list/basic.graphql.json`
- Create: `testdata/pr-list/basic.expected.json`
- Create: `testdata/pr-detail/basic.graphql.json`
- Create: `testdata/pr-detail/basic.expected.json`

- [ ] **Step 1: Create fixture directories**

Run: `mkdir -p testdata/pr-count testdata/pr-list testdata/pr-detail`

Expected: directories exist.

- [ ] **Step 2: Add `pr-count` GraphQL fixture**

Create `testdata/pr-count/basic.graphql.json`:

```json
{
  "data": {
    "repository": {
      "pullRequests": {
        "totalCount": 42
      }
    }
  }
}
```

- [ ] **Step 3: Add `pr-count` expected fixture**

Create `testdata/pr-count/basic.expected.json` with the same content as `basic.graphql.json`.

- [ ] **Step 4: Add `pr-list` GraphQL fixture**

Create `testdata/pr-list/basic.graphql.json`:

```json
{
  "data": {
    "repository": {
      "pullRequests": {
        "nodes": [
          {
            "number": 123,
            "createdAt": "2026-05-01T10:00:00Z",
            "state": "OPEN",
            "mergedAt": null,
            "changedFiles": 3,
            "reviewDecision": "REVIEW_REQUIRED",
            "comments": { "totalCount": 2 },
            "author": { "login": "alice" },
            "reviewRequests": {
              "nodes": [
                { "requestedReviewer": { "login": "bob" } },
                { "requestedReviewer": { "slug": "platform" } }
              ]
            },
            "reviews": {
              "nodes": [
                { "author": { "login": "carol" } },
                { "author": null }
              ]
            }
          }
        ],
        "pageInfo": {
          "hasNextPage": true,
          "endCursor": "cursor-1"
        }
      }
    }
  }
}
```

- [ ] **Step 5: Add `pr-list` expected fixture**

Create `testdata/pr-list/basic.expected.json` with the same content as `basic.graphql.json`.

- [ ] **Step 6: Add `pr-detail` GraphQL fixture**

Create `testdata/pr-detail/basic.graphql.json`:

```json
{
  "data": {
    "repository": {
      "pullRequest": {
        "number": 123,
        "title": "Improve parser",
        "bodyText": "This changes parser behavior.",
        "reviewDecision": "CHANGES_REQUESTED",
        "author": { "login": "alice" },
        "mergeCommit": { "oid": "merge-oid" },
        "baseRefOid": "base-oid",
        "headRefOid": "head-oid",
        "additions": 10,
        "deletions": 4,
        "changedFiles": 3,
        "reviews": {
          "nodes": [
            {
              "id": "review-1",
              "author": { "login": "bob" },
              "state": "CHANGES_REQUESTED",
              "bodyText": "Please simplify this.",
              "submittedAt": "2026-05-01T10:10:00Z",
              "commit": { "oid": "commit-review" }
            }
          ]
        },
        "reviewThreads": {
          "nodes": [
            {
              "isResolved": true,
              "comments": {
                "nodes": [
                  {
                    "id": "comment-1",
                    "author": { "login": "carol" },
                    "bodyText": "This branch is hard to read.",
                    "path": "src/parser.ts",
                    "createdAt": "2026-05-01T10:05:00Z",
                    "line": 12,
                    "originalLine": 10,
                    "startLine": null,
                    "originalStartLine": null,
                    "side": "RIGHT",
                    "startSide": null,
                    "commit": { "oid": "commit-comment" },
                    "originalCommit": { "oid": "commit-original" }
                  }
                ]
              }
            }
          ]
        },
        "files": {
          "totalCount": 3,
          "nodes": [
            { "path": "src/parser.ts", "additions": 8, "deletions": 3, "changeType": "MODIFIED" },
            { "path": "dist/bundle.js", "additions": 100, "deletions": 0, "changeType": "ADDED" },
            { "path": "assets/logo.png", "additions": 0, "deletions": 0, "changeType": "MODIFIED" }
          ],
          "pageInfo": {
            "hasNextPage": false,
            "endCursor": null
          }
        },
        "commits": {
          "nodes": [
            {
              "commit": {
                "oid": "commit-original",
                "messageHeadline": "Initial parser change",
                "committedDate": "2026-05-01T09:00:00Z"
              }
            },
            {
              "commit": {
                "oid": "commit-review",
                "messageHeadline": "Apply review feedback",
                "committedDate": "2026-05-01T11:00:00Z"
              }
            }
          ]
        }
      }
    }
  }
}
```

- [ ] **Step 7: Add `pr-detail` expected fixture**

Create `testdata/pr-detail/basic.expected.json`:

```json
{
  "data": {
    "repository": {
      "pullRequest": {
        "number": 123,
        "title": "Improve parser",
        "description": "This changes parser behavior.",
        "reviewDecision": "CHANGES_REQUESTED",
        "authorLogin": "alice",
        "mergeCommitOid": "merge-oid",
        "reviewStartCommitOid": "commit-original",
        "reviewStartConfidence": "high",
        "codeDiff": {
          "stats": {
            "changedFiles": 3,
            "additions": 10,
            "deletions": 4
          },
          "files": [
            {
              "path": "src/parser.ts",
              "changeType": "MODIFIED",
              "additions": 8,
              "deletions": 3
            }
          ],
          "excludedFiles": [
            { "path": "dist/bundle.js", "reason": "likely-generated" },
            { "path": "assets/logo.png", "reason": "likely-binary" }
          ],
          "filePageInfo": {
            "hasNextPage": false,
            "endCursor": null,
            "totalCount": 3
          },
          "strategy": {
            "baseCommit": "base-oid",
            "headCommit": "head-oid"
          }
        },
        "conversations": {
          "reviews": [
            {
              "authorLogin": "bob",
              "state": "CHANGES_REQUESTED",
              "body": "Please simplify this.",
              "submittedAt": "2026-05-01T10:10:00Z",
              "commitOid": "commit-review"
            }
          ],
          "reviewThreads": [
            {
              "isResolved": true,
              "comments": [
                {
                  "id": "comment-1",
                  "authorLogin": "carol",
                  "body": "This branch is hard to read.",
                  "path": "src/parser.ts",
                  "createdAt": "2026-05-01T10:05:00Z",
                  "line": 12,
                  "originalLine": 10,
                  "startLine": null,
                  "originalStartLine": null,
                  "side": "RIGHT",
                  "startSide": null,
                  "commitOid": "commit-comment",
                  "originalCommitOid": "commit-original"
                }
              ]
            }
          ]
        },
        "commits": [
          {
            "oid": "commit-original",
            "messageHeadline": "Initial parser change",
            "committedDate": "2026-05-01T09:00:00Z"
          },
          {
            "oid": "commit-review",
            "messageHeadline": "Apply review feedback",
            "committedDate": "2026-05-01T11:00:00Z"
          }
        ]
      }
    }
  }
}
```

- [ ] **Step 8: Commit fixtures**

Run only if committing is requested for implementation work:

```bash
git add testdata
git commit -m "test: add GraphQL parity fixtures"
```

## Task 2: Extract Bun Pure Functions And Add Fixture Tests

**Files:**
- Create: `src/cli/pr-count/query.ts`
- Create: `src/cli/pr-list/query.ts`
- Create: `src/cli/pr-detail/transform.ts`
- Modify: `src/cli/pr-count/index.ts`
- Modify: `src/cli/pr-list/index.ts`
- Modify: `src/cli/pr-detail/index.ts`
- Create: `src/cli/pr-count/fixture.test.ts`
- Create: `src/cli/pr-list/fixture.test.ts`
- Create: `src/cli/pr-detail/fixture.test.ts`

- [ ] **Step 1: Extract `pr-count` query function**

Create `src/cli/pr-count/query.ts`:

```ts
import { runGraphQlCommandParsed } from '../../core';
import { GRAPHQL_QUERY } from './graphql-query';
import { type CliArgs, type PrCountOutput, prCountOutputSchema } from './schema';

export function buildFilters(args: CliArgs): string[] {
  return (Object.keys(args) as Array<keyof CliArgs>).flatMap((key) => {
    const value = args[key];
    return value !== undefined ? [`${key}=${value}`] : [];
  });
}

export function queryPullRequestCount(args: CliArgs): PrCountOutput {
  return runGraphQlCommandParsed({
    query: GRAPHQL_QUERY,
    filters: buildFilters(args),
    outputSchema: prCountOutputSchema,
  });
}
```

- [ ] **Step 2: Update `pr-count` entrypoint**

Modify `src/cli/pr-count/index.ts`:

```ts
import { runCli } from '../../core';
import { queryPullRequestCount } from './query';
import { cliArgsSchema } from './schema';
import { USAGE } from './usage';

runCli({
  usage: USAGE,
  cliArgsSchema,
  execute: queryPullRequestCount,
});
```

- [ ] **Step 3: Extract `pr-list` query function**

Create `src/cli/pr-list/query.ts`:

```ts
import { runGraphQlCommandParsed } from '../../core';
import { GRAPHQL_QUERY } from './graphql-query';
import { type CliArgs, type PrListOutput, prListOutputSchema } from './schema';

export function buildFilters(args: CliArgs): string[] {
  return (Object.keys(args) as Array<keyof CliArgs>).flatMap((key) => {
    const value = args[key];
    return value !== undefined ? [`${key}=${value}`] : [];
  });
}

export function queryPullRequestList(args: CliArgs): PrListOutput {
  return runGraphQlCommandParsed({
    query: GRAPHQL_QUERY,
    filters: buildFilters(args),
    outputSchema: prListOutputSchema,
  });
}
```

- [ ] **Step 4: Update `pr-list` entrypoint**

Modify `src/cli/pr-list/index.ts`:

```ts
import { runCli } from '../../core';
import { queryPullRequestList } from './query';
import { cliArgsSchema } from './schema';
import { USAGE } from './usage';

runCli({
  usage: USAGE,
  cliArgsSchema,
  execute: queryPullRequestList,
});
```

- [ ] **Step 5: Extract `pr-detail` transform function**

Create `src/cli/pr-detail/transform.ts` by moving `PrDetailPayload`, `ReviewStartConfidence`, `getReviewStartAnchor`, and `transformOutput` from `src/cli/pr-detail/index.ts`. Export `transformOutput`.

- [ ] **Step 6: Update `pr-detail` entrypoint**

Modify `src/cli/pr-detail/index.ts` so it keeps only imports, `queryPullRequestDetail`, and `runCli`:

```ts
import { runCli, runGraphQlCommandParsed } from '../../core';
import { GRAPHQL_QUERY } from './graphql-query';
import {
  type CliArgs,
  type PrDetailOutput,
  cliArgsSchema,
  prDetailGraphQlOutputSchema,
  prDetailOutputSchema,
} from './schema';
import { transformOutput } from './transform';
import { USAGE } from './usage';

function queryPullRequestDetail(args: CliArgs): PrDetailOutput {
  const graphQlOutput = runGraphQlCommandParsed({
    query: GRAPHQL_QUERY,
    filters: (Object.keys(args) as Array<keyof CliArgs>).flatMap((key) => {
      const value = args[key];
      return value !== undefined ? [`${key}=${value}`] : [];
    }),
    outputSchema: prDetailGraphQlOutputSchema,
  });

  const transformed = transformOutput(graphQlOutput);

  return prDetailOutputSchema.parse(transformed);
}

runCli({
  usage: USAGE,
  cliArgsSchema,
  execute: queryPullRequestDetail,
});
```

- [ ] **Step 7: Add Bun fixture tests**

Create `src/cli/pr-detail/fixture.test.ts`:

```ts
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'bun:test';
import { prDetailGraphQlOutputSchema } from './schema';
import { transformOutput } from './transform';

function readJson(path: string): unknown {
  return JSON.parse(readFileSync(resolve(import.meta.dir, '../../../', path), 'utf8'));
}

describe('pr-detail fixture parity', () => {
  it('transforms GraphQL response to the current output contract', () => {
    const input = prDetailGraphQlOutputSchema.parse(readJson('testdata/pr-detail/basic.graphql.json'));
    const expected = readJson('testdata/pr-detail/basic.expected.json');

    expect(transformOutput(input)).toEqual(expected);
  });
});
```

Create `src/cli/pr-count/fixture.test.ts`:

```ts
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'bun:test';
import { prCountOutputSchema } from './schema';

function readJson(path: string): unknown {
  return JSON.parse(readFileSync(resolve(import.meta.dir, '../../../', path), 'utf8'));
}

describe('pr-count fixture parity', () => {
  it('keeps the current output contract', () => {
    const input = prCountOutputSchema.parse(readJson('testdata/pr-count/basic.graphql.json'));
    const expected = readJson('testdata/pr-count/basic.expected.json');

    expect(input).toEqual(expected);
  });
});
```

Create `src/cli/pr-list/fixture.test.ts`:

```ts
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'bun:test';
import { prListOutputSchema } from './schema';

function readJson(path: string): unknown {
  return JSON.parse(readFileSync(resolve(import.meta.dir, '../../../', path), 'utf8'));
}

describe('pr-list fixture parity', () => {
  it('keeps the current output contract', () => {
    const input = prListOutputSchema.parse(readJson('testdata/pr-list/basic.graphql.json'));
    const expected = readJson('testdata/pr-list/basic.expected.json');

    expect(input).toEqual(expected);
  });
});
```

- [ ] **Step 8: Run Bun tests**

Run: `bun test src/cli/pr-count/fixture.test.ts src/cli/pr-list/fixture.test.ts src/cli/pr-detail/fixture.test.ts`

Expected: all fixture tests pass.

## Task 3: Add Go Module, Tool Pin, And CLI Skeleton

**Files:**
- Modify: `.mise/config.toml`
- Create: `go.mod`
- Create: `cmd/gh-usecase/main.go`
- Create: `internal/cli/args.go`
- Create: `internal/cli/runner.go`
- Create: `internal/cli/usage.go`

- [ ] **Step 1: Pin Go 1.26.x**

Run: `mise ls-remote go@1.26`

Expected: prints available `1.26.x` versions. Choose the highest patch version.

Modify `.mise/config.toml` so it contains Bun and the exact selected Go version:

```toml
[tools]
bun = "1.3.9"
go = "1.26.0"
```

If `mise ls-remote go@1.26` reports a higher patch than `1.26.0`, use that exact patch instead.

- [ ] **Step 2: Create Go module**

Run: `go mod init github.com/y-writings/gh-usecase`

Expected: `go.mod` is created.

- [ ] **Step 3: Add Go CLI parser**

Create `internal/cli/args.go`:

```go
package cli

type ParsedArgs struct {
	Options       map[string]string
	Positionals    []string
	HelpRequested bool
}

func ParseArgs(argv []string) ParsedArgs {
	options := map[string]string{}
	positionals := []string{}
	helpRequested := false

	for i := 0; i < len(argv); i++ {
		token := argv[i]
		if token == "--help" || token == "-h" {
			helpRequested = true
			continue
		}
		if len(token) > 2 && token[:2] == "--" {
			withoutPrefix := token[2:]
			if withoutPrefix == "" {
				continue
			}
			if key, value, ok := splitEquals(withoutPrefix); ok {
				if key != "" {
					options[key] = value
				}
				continue
			}
			if i+1 < len(argv) && !isLongOption(argv[i+1]) {
				options[withoutPrefix] = argv[i+1]
				i++
			}
			continue
		}
		positionals = append(positionals, token)
	}

	return ParsedArgs{Options: options, Positionals: positionals, HelpRequested: helpRequested}
}

func splitEquals(value string) (string, string, bool) {
	for i, char := range value {
		if char == '=' {
			return value[:i], value[i+1:], true
		}
	}
	return "", "", false
}

func isLongOption(value string) bool {
	return len(value) >= 2 && value[:2] == "--"
}
```

- [ ] **Step 4: Add usage text**

Create `internal/cli/usage.go` with root and command usage strings for `gh-usecase`, `pr-count`, `pr-list`, and `pr-detail`.

- [ ] **Step 5: Add runner skeleton**

Create `internal/cli/runner.go` with a `Run(argv []string, stdout io.Writer, stderr io.Writer) int` function that handles root help and unknown commands. Command execution can return `unknown command` until command packages are wired.

- [ ] **Step 6: Add main**

Create `cmd/gh-usecase/main.go`:

```go
package main

import (
	"os"

	"github.com/y-writings/gh-usecase/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
```

- [ ] **Step 7: Verify skeleton**

Run: `go run ./cmd/gh-usecase --help`

Expected: root usage is printed and exit code is `0`.

## Task 4: Add GitHub API Boundary

**Files:**
- Create: `internal/githubapi/client.go`
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Add dependency**

Run: `go get github.com/cli/go-gh/v2/pkg/api`

Expected: `go.mod` and `go.sum` include `github.com/cli/go-gh/v2`.

- [ ] **Step 2: Create GraphQL client boundary**

Create `internal/githubapi/client.go`:

```go
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
```

- [ ] **Step 3: Verify package compiles**

Run: `go test ./internal/githubapi`

Expected: package compiles and tests report no test files or pass.

## Task 5: Port `pr-count`

**Files:**
- Create: `internal/prcount/query.go`
- Create: `internal/prcount/types.go`
- Create: `internal/prcount/prcount.go`
- Create: `internal/prcount/prcount_test.go`
- Modify: `internal/cli/runner.go`

- [ ] **Step 1: Add failing Go fixture test**

Create `internal/prcount/prcount_test.go` with a fake client that reads `testdata/pr-count/basic.graphql.json`, verifies `state` becomes `[]string{"OPEN"}`, calls `prcount.Execute`, and compares output to `testdata/pr-count/basic.expected.json`.

- [ ] **Step 2: Run failing test**

Run: `go test ./internal/prcount`

Expected: FAIL because `Execute` is not defined.

- [ ] **Step 3: Implement query and types**

Create `internal/prcount/query.go`:

```go
package prcount

const graphQLQuery = `query ($owner: String!, $name: String!, $state: [PullRequestState!]) {
        repository(owner: $owner, name: $name) {
            pullRequests(states: $state) {
                totalCount
            }
        }
    }`
```

Create `internal/prcount/types.go` with `Input`, `Output`, `Data`, `Repository`, and `PullRequests` structs that encode to `data.repository.pullRequests.totalCount`.

- [ ] **Step 4: Implement execution**

Create `internal/prcount/prcount.go` with input validation for `owner`, `name`, and optional `state`; build variables with `state` normalized to `[]string`; call `GraphQLClient.DoWithContext`; return the decoded response unchanged.

- [ ] **Step 5: Wire command into runner**

Modify `internal/cli/runner.go` so `pr-count` parses args, creates the default GraphQL client, calls `prcount.Execute`, and writes pretty JSON to stdout.

- [ ] **Step 6: Verify `pr-count` tests**

Run: `go test ./internal/prcount ./internal/cli`

Expected: PASS.

## Task 6: Port `pr-list`

**Files:**
- Create: `internal/prlist/query.go`
- Create: `internal/prlist/types.go`
- Create: `internal/prlist/prlist.go`
- Create: `internal/prlist/prlist_test.go`
- Modify: `internal/cli/runner.go`

- [ ] **Step 1: Add failing Go fixture test**

Create `internal/prlist/prlist_test.go` with a fake client that reads `testdata/pr-list/basic.graphql.json`, verifies `first` defaults to `30`, verifies `state` becomes `[]string{"OPEN"}` when provided, and compares output to `testdata/pr-list/basic.expected.json`.

- [ ] **Step 2: Run failing test**

Run: `go test ./internal/prlist`

Expected: FAIL because `Execute` is not defined.

- [ ] **Step 3: Implement query, types, and execution**

Create `internal/prlist/query.go`:

```go
package prlist

const graphQLQuery = `query ($owner: String!, $name: String!, $state: [PullRequestState!], $first: Int!, $after: String) {
    repository(owner: $owner, name: $name) {
      pullRequests(states: $state, first: $first, after: $after, orderBy: {field: CREATED_AT, direction: DESC}) {
        nodes {
          number
          createdAt
          state
          mergedAt
          changedFiles
          reviewDecision
          comments {
            totalCount
          }
          author {
            login
          }
          reviewRequests(first: 20) {
            nodes {
              requestedReviewer {
                ... on User {
                  login
                }
                ... on Team {
                  slug
                }
                ... on Bot {
                  login
                }
                ... on Mannequin {
                  login
                }
              }
            }
          }
          reviews(first: 20) {
            nodes {
              author {
                login
              }
            }
          }
        }
        pageInfo {
          hasNextPage
          endCursor
        }
      }
    }
  }`
```

Create `internal/prlist/types.go` with structs for `nodes`, `pageInfo`, author, reviews, comments, and review requests.

Create `internal/prlist/prlist.go` with validation for `owner`, `name`, optional `state`, optional `after`, and `first` in `1..100` with default `30`.

- [ ] **Step 4: Wire command into runner**

Modify `internal/cli/runner.go` so `pr-list` executes through `prlist.Execute` and writes pretty JSON.

- [ ] **Step 5: Verify `pr-list` tests**

Run: `go test ./internal/prlist ./internal/cli`

Expected: PASS.

## Task 7: Port `pr-detail`

**Files:**
- Create: `internal/prdetail/query.go`
- Create: `internal/prdetail/types.go`
- Create: `internal/prdetail/file_utils.go`
- Create: `internal/prdetail/prdetail.go`
- Create: `internal/prdetail/prdetail_test.go`
- Modify: `internal/cli/runner.go`

- [ ] **Step 1: Add failing Go fixture test**

Create `internal/prdetail/prdetail_test.go` with a fake client that reads `testdata/pr-detail/basic.graphql.json`, verifies `filesFirst` defaults to `40`, calls `prdetail.Execute`, and compares output to `testdata/pr-detail/basic.expected.json`.

- [ ] **Step 2: Run failing test**

Run: `go test ./internal/prdetail`

Expected: FAIL because `Execute` is not defined.

- [ ] **Step 3: Implement query and GraphQL types**

Create `internal/prdetail/query.go`:

```go
package prdetail

const graphQLQuery = `query ($owner: String!, $name: String!, $number: Int!, $filesFirst: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      number
      title
      bodyText
      reviewDecision
      author {
        login
      }
      mergeCommit {
        oid
      }
      baseRefOid
      headRefOid
      additions
      deletions
      changedFiles
      reviews(first: 100) {
        nodes {
          id
          author {
            login
          }
          state
          bodyText
          submittedAt
          commit {
            oid
          }
        }
      }
      reviewThreads(first: 100) {
        nodes {
          isResolved
          comments(first: 100) {
            nodes {
              id
              author {
                login
              }
              bodyText
              path
              createdAt
              line
              originalLine
              startLine
              originalStartLine
              side
              startSide
              commit {
                oid
              }
              originalCommit {
                oid
              }
            }
          }
        }
      }
      files(first: $filesFirst) {
        totalCount
        nodes {
          path
          additions
          deletions
          changeType
        }
        pageInfo {
          hasNextPage
          endCursor
        }
      }
      commits(first: 100) {
        nodes {
          commit {
            oid
            messageHeadline
            committedDate
          }
        }
      }
    }
  }
}`
```

Create `internal/prdetail/types.go` with separate GraphQL response structs and final output structs. Use pointers for nullable fields such as author, merge commit, review commit, comment lines, sides, and pull request.

- [ ] **Step 4: Implement file utility parity**

Create `internal/prdetail/file_utils.go` with these generated path regexes: `(^|/)dist/`, `(^|/)build/`, `(^|/)coverage/`, `(^|/)vendor/`, `(^|/)generated/`, `(^|/)__snapshots__/`, `\.min\.[a-z0-9]+$`, `\.lock$`, `^pnpm-lock\.yaml$`, `^bun\.lockb$`, `^yarn\.lock$`, `^package-lock\.json$`, `^Cargo\.lock$`. Use this binary extension set: `png`, `jpg`, `jpeg`, `gif`, `webp`, `bmp`, `ico`, `svg`, `pdf`, `zip`, `gz`, `tar`, `rar`, `7z`, `mp3`, `mp4`, `mov`, `avi`, `wav`, `ogg`, `ttf`, `otf`, `woff`, `woff2`, `eot`, `jar`, `exe`, `dll`, `so`, `dylib`, `class`.

- [ ] **Step 5: Implement transform parity**

Create `internal/prdetail/prdetail.go` with validation for `owner`, `name`, positive integer `number`, `filesFirst` in `1..100` with default `40`; execute GraphQL; transform output to the shape in `testdata/pr-detail/basic.expected.json`, including review start anchor selection and excluded file reasons.

- [ ] **Step 6: Wire command into runner**

Modify `internal/cli/runner.go` so `pr-detail` executes through `prdetail.Execute` and writes pretty JSON.

- [ ] **Step 7: Verify `pr-detail` tests**

Run: `go test ./internal/prdetail ./internal/cli`

Expected: PASS.

## Task 8: Add CLI Contract Tests

**Files:**
- Create: `internal/cli/args_test.go`
- Create: `internal/cli/runner_test.go`

- [ ] **Step 1: Test parser parity**

Create `internal/cli/args_test.go` covering `--key value`, `--key=value`, `--help`, `-h`, ignored positionals, ignored unknown options, and missing option values.

- [ ] **Step 2: Test runner output contract**

Create `internal/cli/runner_test.go` covering root help exit `0`, unknown command non-zero, and validation failure non-zero with usage on stderr. Successful command JSON output is covered by command package fixture tests in Tasks 5-7.

- [ ] **Step 3: Run CLI tests**

Run: `go test ./internal/cli`

Expected: PASS.

## Task 9: Final Verification

**Files:**
- Modify only if verification exposes formatting, lint, or test failures.

- [ ] **Step 1: Format Go code**

Run: `gofmt -w cmd internal`

Expected: no output.

- [ ] **Step 2: Run Go tests**

Run: `go test ./...`

Expected: PASS.

- [ ] **Step 3: Run Bun tests**

Run: `bun test src/cli/pr-count/fixture.test.ts src/cli/pr-list/fixture.test.ts src/cli/pr-detail/fixture.test.ts`

Expected: PASS.

- [ ] **Step 4: Run existing TypeScript checks**

Run: `bun run typecheck`

Expected: PASS.

- [ ] **Step 5: Run existing lint/check**

Run: `bun run check`

Expected: PASS.

- [ ] **Step 6: Manual smoke test if credentials are available**

Run: `GH_TOKEN=$GH_TOKEN go run ./cmd/gh-usecase pr-count --owner octokit --name rest.js --state OPEN`

Expected: pretty JSON stdout with `data.repository.pullRequests.totalCount` and exit code `0`.

- [ ] **Step 7: Review worktree**

Run: `git status --short`

Expected: only intended files are modified or created.

- [ ] **Step 8: Commit implementation if requested**

Run only if committing is requested:

```bash
git add .mise/config.toml go.mod go.sum cmd internal src testdata docs/superpowers
git commit -m "feat: add initial Go CLI port"
```

## Self-Review Notes

- Spec coverage: the plan covers shared fixtures, Bun refactor, Go module/tool pin, Go CLI architecture, GitHub API boundary, all three commands, CLI parser parity, output/error verification, and offline tests.
- Placeholder scan: no incomplete task or deferred implementation requirement remains.
- Type consistency: `GraphQLClient.DoWithContext`, `Execute`, `ParseArgs`, `Run`, and command names are used consistently across tasks.
