package cli

const RootUsage = `Usage: gh-usecase <command> [options]

Commands:
  pr-count    Fetch pull request total count
  pr-list     Fetch pull request list
  pr-detail   Fetch pull request detail for analysis`

const PrCountUsage = `Usage: gh-usecase pr-count --owner <owner> --name <name> [--state OPEN|CLOSED|MERGED]

Fetch pull request total count.`

const PrListUsage = `Usage: gh-usecase pr-list --owner <owner> --name <name> [--state OPEN|CLOSED|MERGED] [--first <1-100>] [--after <cursor>]

Fetch pull request list.`

const PrDetailUsage = `Usage: gh-usecase pr-detail --owner <owner> --name <name> --number <number> [--filesFirst <1-100>]

Fetch pull request detail for analysis.`
