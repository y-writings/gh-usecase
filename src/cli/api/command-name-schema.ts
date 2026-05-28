import { z } from 'zod';

const commandDefinitionsSchema = z.object({
  'pr-count': z.string().describe('Fetch pull request total count'),
  'pr-list': z.string().describe('Fetch pull request list'),
  'pr-detail': z.string().describe('Fetch pull request detail for analysis'),
});

export const commandNameSchema = commandDefinitionsSchema.keyof();

const commandNames = commandNameSchema.options;

export function getCommandUsageLines(): string[] {
  const longestCommandLength = Math.max(...commandNames.map((command) => command.length));

  return commandNames.map((commandName) => {
    const command = commandName.padEnd(longestCommandLength);
    const description = commandDefinitionsSchema.shape[commandName].description ?? '';

    return `${command}   ${description}`.trimEnd();
  });
}

export type CommandName = z.infer<typeof commandNameSchema>;
