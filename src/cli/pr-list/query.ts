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
