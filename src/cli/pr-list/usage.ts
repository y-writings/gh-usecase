export const USAGE = `Usage:
  bun run src/cli/pr-list/index.ts --owner <owner> --name <repo-name> [--state <OPEN|CLOSED|MERGED>] [--after <cursor>] [--first <1-100>]

Examples:
  bun run src/cli/pr-list/index.ts --owner octokit --name rest.js --state OPEN
  bun run src/cli/pr-list/index.ts --owner vercel --name next.js --state CLOSED --first 20
  bun run src/cli/pr-list/index.ts --owner vercel --name next.js --state MERGED --after Y3Vyc29yOnYyOpHOUH8B7A== --first 20
`;
