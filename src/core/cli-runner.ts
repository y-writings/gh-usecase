import type { ZodType, ZodTypeDef } from 'zod';
import { parseCliInput } from './cli-args';
import { exitWithUsage } from './cli-usage';
import { CommandExecutionError } from './command-runner';
import { ResponseParseError } from './response-parser';

export interface CliRunnerConfig<TArgs, TOutput> {
  usage: string;
  cliArgsSchema: ZodType<TArgs, ZodTypeDef, unknown>;
  execute: (args: TArgs) => TOutput;
  unknownExitCode?: number;
}

function parseArgs<TArgs>(
  argv: string[],
  usage: string,
  cliArgsSchema: ZodType<TArgs, ZodTypeDef, unknown>,
): TArgs {
  const parsed = parseCliInput(argv);

  if (parsed.helpRequested) {
    exitWithUsage({ usage, exitCode: 0 });
  }

  const result = cliArgsSchema.safeParse(parsed.options);
  if (!result.success) {
    const details = result.error.issues.map((issue) => issue.message).join(', ');
    exitWithUsage({ usage, message: details });
  }

  return result.data;
}

function formatCommandError(error: CommandExecutionError): string {
  return `Failed to execute command: ${error.stderr || error.message}`;
}

function formatParseError(error: ResponseParseError): string {
  return `Failed to parse gh response: ${error.message}`;
}

function formatUnknownError(error: unknown): string {
  if (error instanceof Error) {
    return `Failed to execute command: ${error.message}`;
  }

  return `Failed to execute command: ${String(error)}`;
}

export function runCli<TArgs, TOutput>(config: CliRunnerConfig<TArgs, TOutput>): void {
  const args = parseArgs(process.argv.slice(2), config.usage, config.cliArgsSchema);

  try {
    const output = config.execute(args);
    console.log(JSON.stringify(output, null, 2));
  } catch (error) {
    if (error instanceof CommandExecutionError) {
      console.error(formatCommandError(error));
      process.exit(1);
    }

    if (error instanceof ResponseParseError) {
      console.error(formatParseError(error));
      process.exit(1);
    }

    const message = formatUnknownError(error);
    console.error(message);
    process.exit(config.unknownExitCode ?? 1);
  }
}
