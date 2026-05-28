import { z } from 'zod';
import { nameParamSchema, ownerParamSchema, stateParamSchema } from '../api/input-params-schema';

export const cliArgsSchema = z.object({
  owner: ownerParamSchema,
  name: nameParamSchema,
  state: stateParamSchema,
});

export type CliArgs = z.infer<typeof cliArgsSchema>;

export const prCountOutputSchema = z.object({
  data: z.object({
    repository: z
      .object({
        pullRequests: z.object({
          totalCount: z.number().int().nonnegative(),
        }),
      })
      .strict(),
  }),
});

export type PrCountOutput = z.infer<typeof prCountOutputSchema>;
