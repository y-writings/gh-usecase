package cli

const RootUsage = `Usage: gh-usecase <command> [options]

Commands:
  pr-count              Fetch pull request total count
  pr-list               Fetch pull request list
  pr-detail             Fetch pull request detail for analysis
  codeql-default-setup  Configure CodeQL default setup for a repository`

const PrCountUsage = `Usage: gh-usecase pr-count --owner <owner> --name <name> [--state OPEN|CLOSED|MERGED]

Fetch pull request total count.`

const PrListUsage = `Usage: gh-usecase pr-list --owner <owner> --name <name> [--state OPEN|CLOSED|MERGED] [--first <1-100>] [--after <cursor>]

Fetch pull request list.`

const PrDetailUsage = `Usage: gh-usecase pr-detail --owner <owner> --name <name> --number <number> [--filesFirst <1-100>]

Fetch pull request detail for analysis.`

const CodeQLDefaultSetupUsage = `Usage: gh-usecase codeql-default-setup --owner <owner> --repo <repo> --languages <csv>

Configure CodeQL default setup with runner_type=standard, query_suite=default, and threat_model=remote.

Languages must be a comma-separated list containing only: actions, c-cpp, csharp, go, java-kotlin, javascript-typescript, python, ruby, swift.`
