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
