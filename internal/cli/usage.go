package cli

const RootUsage = `Usage: gh-usecase <command> [options]

Commands:
  pr-count                      Fetch pull request total count
  pr-list                       Fetch pull request list
  repo-list                     Fetch repositories owned by an account
  pr-detail                     Fetch pull request detail for analysis
  codeql-default-setup          Configure CodeQL default setup for a repository
  pull-request-creation-policy  Configure who can create pull requests for a repository`

const PrCountUsage = `Usage: gh-usecase pr-count --owner <owner> --name <name> [--state OPEN|CLOSED|MERGED]

Fetch pull request total count.`

const PrListUsage = `Usage: gh-usecase pr-list --owner <owner> --name <name> [--state OPEN|CLOSED|MERGED] [--first <1-100>] [--after <cursor>]

Fetch pull request list.`

const RepoListUsage = `Usage: gh-usecase repo-list --owner <account> [--first <1-100>] [--after <cursor>]

Fetch one name-ordered page of repositories owned by a user or organization and visible to the authenticated user.`

const PrDetailUsage = `Usage: gh-usecase pr-detail --owner <owner> --name <name> --number <number> [--filesFirst <1-100>]

Fetch pull request detail for analysis.`

const CodeQLDefaultSetupUsage = `Usage: gh-usecase codeql-default-setup --owner <owner> --repo <repo> --languages <csv>

Configure CodeQL default setup with runner_type=standard, query_suite=default, and threat_model=remote.

Languages must be a comma-separated list containing only: actions, c-cpp, csharp, go, java-kotlin, javascript-typescript, python, ruby, swift.`

const PullRequestCreationPolicyUsage = `Usage: gh-usecase pull-request-creation-policy --owner <owner> --repo <repo> --policy <all|collaborators_only>

Configure who can create pull requests for a repository.

Policy must be one of: all, collaborators_only.`
