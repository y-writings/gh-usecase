import { runCli } from '../../core';
import { queryPullRequestCount } from './query';
import { cliArgsSchema } from './schema';
import { USAGE } from './usage';

runCli({
  usage: USAGE,
  cliArgsSchema,
  execute: queryPullRequestCount,
});
