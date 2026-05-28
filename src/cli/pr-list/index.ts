import { runCli, runGraphQlCommandParsed } from '../../core';
import { GRAPHQL_QUERY } from './graphql-query';
import { type CliArgs, type PrListOutput, cliArgsSchema, prListOutputSchema } from './schema';
import { USAGE } from './usage';

function queryPullRequestList(args: CliArgs): PrListOutput {
  const filters = (Object.keys(args) as Array<keyof CliArgs>).flatMap((key) => {
    const value = args[key];
    return value !== undefined ? [`${key}=${value}`] : [];
  });

  return runGraphQlCommandParsed({
    query: GRAPHQL_QUERY,
    filters,
    outputSchema: prListOutputSchema,
  });
}

runCli({
  usage: USAGE,
  cliArgsSchema,
  execute: queryPullRequestList,
});
