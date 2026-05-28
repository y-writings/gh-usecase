#!/usr/bin/env bun

import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { commandNameSchema, getCommandUsageLines } from './cli/api/command-name-schema';
import { exitWithUsage } from './core';

const commandUsageLines = getCommandUsageLines()
  .map((line) => `  ${line}`)
  .join('\n');

const moduleDirectory = dirname(fileURLToPath(import.meta.url));

const USAGE = `Usage:
  gh-pr-counter <command> [options]

Commands:
${commandUsageLines}
`;

function main(): void {
  const [command] = process.argv.slice(2);

  if (!command || command === '--help' || command === '-h') {
    exitWithUsage({ usage: USAGE, exitCode: 0 });
  }

  const parsedCommand = commandNameSchema.safeParse(command);
  if (!parsedCommand.success) {
    exitWithUsage({ usage: USAGE, message: `unknown command '${command}'` });
  }

  try {
    const commandScriptPath = resolve(moduleDirectory, `cli/${parsedCommand.data}/index.ts`);
    const result = Bun.spawnSync(['bun', 'run', commandScriptPath, ...process.argv.slice(3)], {
      stdin: 'inherit',
      stdout: 'inherit',
      stderr: 'inherit',
    });

    if (result.exitCode !== 0) {
      process.exit(result.exitCode);
    }
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    console.error(`Failed to execute command: ${message}`);
    process.exit(1);
  }
}

main();
