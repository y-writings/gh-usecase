import { runCli } from '../../core';
import { queryPullRequestList } from './query';
import { cliArgsSchema } from './schema';
import { USAGE } from './usage';

runCli({
  usage: USAGE,
  cliArgsSchema,
  execute: queryPullRequestList,
});
