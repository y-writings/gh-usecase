import type { ZodTypeAny } from 'zod';
import { ResponseParseError } from './response-parser';
import { parseResponseTextWithSchema } from './response-parser';

type StreamMode = 'inherit' | 'ignore' | 'pipe';

export type CommandOptions = {
  stdin?: StreamMode;
  stdout?: StreamMode;
  stderr?: StreamMode;
};

type RunGraphQlCommandParsedInput<TSchema extends ZodTypeAny> = {
  query: string;
  filters: readonly string[];
  outputSchema: TSchema;
  commandOptions?: CommandOptions;
};

export class CommandExecutionError extends Error {
  command: readonly string[];
  exitCode: number;
  stderr: string;

  constructor(command: readonly string[], exitCode: number, stderr: string) {
    super(`command failed with exit code ${exitCode}`);
    this.name = 'CommandExecutionError';
    this.command = command;
    this.exitCode = exitCode;
    this.stderr = stderr;
  }
}

function buildGraphQlCommand(query: string, filters: readonly string[]): string[] {
  return [
    'gh',
    'api',
    'graphql',
    '-f',
    `query=${query}`,
    ...filters.flatMap((filter) => ['-F', filter]),
  ];
}

export function runGraphQlCommand(
  query: string,
  filters: readonly string[],
  options: CommandOptions = {},
): string {
  const command = buildGraphQlCommand(query, filters);

  const { stdin = 'pipe', stdout = 'pipe', stderr = 'pipe' } = options;

  const result = Bun.spawnSync(command, {
    stdin,
    stdout,
    stderr,
  });

  const stdoutData = stdout === 'pipe' ? new TextDecoder().decode(result.stdout).trim() : '';
  const stderrData = stderr === 'pipe' ? new TextDecoder().decode(result.stderr).trim() : '';

  if (result.exitCode !== 0) {
    throw new CommandExecutionError([...command], result.exitCode, stderrData);
  }

  if (stdout === 'pipe' && !stdoutData) {
    throw new ResponseParseError('empty response from command');
  }

  return stdoutData;
}

export function runGraphQlCommandParsed<TSchema extends ZodTypeAny>({
  query,
  filters,
  outputSchema,
  commandOptions,
}: RunGraphQlCommandParsedInput<TSchema>): TSchema['_output'] {
  return parseResponseTextWithSchema({
    responseText: runGraphQlCommand(query, filters, commandOptions),
    outputSchema,
  });
}
