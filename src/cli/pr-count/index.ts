import { runCli, runGraphQlCommandParsed } from '../../core';
import { GRAPHQL_QUERY } from './graphql-query';
import { type CliArgs, type PrCountOutput, cliArgsSchema, prCountOutputSchema } from './schema';
import { USAGE } from './usage';

function queryPullRequestCount(args: CliArgs): PrCountOutput {
  const filters = (Object.keys(args) as Array<keyof CliArgs>).flatMap((key) => {
    const value = args[key];
    return value !== undefined ? [`${key}=${value}`] : [];
  });

  return runGraphQlCommandParsed({
    query: GRAPHQL_QUERY,
    filters,
    outputSchema: prCountOutputSchema,
  });
}

runCli({
  usage: USAGE,
  cliArgsSchema,
  execute: queryPullRequestCount,
});
