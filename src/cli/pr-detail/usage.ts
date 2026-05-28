export const USAGE = `Usage:
  bun run src/cli/pr-detail/index.ts --owner <owner> --name <repo-name> --number <pull-request-number> [--filesFirst <1-100>]

Examples:
  bun run src/cli/pr-detail/index.ts --owner octokit --name rest.js --number 1
  bun run src/cli/pr-detail/index.ts --owner vercel --name next.js --number 12345 --filesFirst 60
`;
