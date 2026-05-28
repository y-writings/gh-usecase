export const USAGE = `Usage:
  bun run src/cli/pr-count/index.ts --owner <owner> --name <repo-name> [--state <OPEN|CLOSED|MERGED>]

Examples:
  bun run src/cli/pr-count/index.ts --owner octokit --name rest.js --state OPEN
  bun run src/cli/pr-count/index.ts --owner vercel --name next.js --state CLOSED
  bun run src/cli/pr-count/index.ts --owner vercel --name next.js --state MERGED
`;
