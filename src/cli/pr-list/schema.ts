import { z } from 'zod';
import {
  afterParamSchema,
  firstParamSchema,
  nameParamSchema,
  ownerParamSchema,
  stateParamSchema,
} from '../api/input-params-schema';

export const cliArgsSchema = z.object({
  owner: ownerParamSchema,
  name: nameParamSchema,
  state: stateParamSchema,
  after: afterParamSchema,
  first: firstParamSchema,
});

export type CliArgs = z.infer<typeof cliArgsSchema>;

const graphqlPullRequestNodeSchema = z.object({
  number: z.number().int().nonnegative(),
  createdAt: z.string(),
  state: z.enum(['OPEN', 'CLOSED']),
  mergedAt: z.string().nullable(),
  changedFiles: z.number().int().nonnegative(),
  reviewDecision: z.enum(['APPROVED', 'CHANGES_REQUESTED', 'REVIEW_REQUIRED']).nullable(),
  comments: z.object({
    totalCount: z.number().int().nonnegative(),
  }),
  author: z
    .object({
      login: z.string(),
    })
    .nullable(),
  reviewRequests: z.object({
    nodes: z.array(
      z
        .object({
          requestedReviewer: z
            .object({
              login: z.string().optional(),
              slug: z.string().optional(),
            })
            .nullable(),
        })
        .passthrough(),
    ),
  }),
  reviews: z.object({
    nodes: z.array(
      z.object({
        author: z
          .object({
            login: z.string(),
          })
          .nullable(),
      }),
    ),
  }),
});

export const prListGraphQlOutputSchema = z.object({
  data: z.object({
    repository: z
      .object({
        pullRequests: z.object({
          nodes: z.array(graphqlPullRequestNodeSchema),
          pageInfo: z.object({
            hasNextPage: z.boolean(),
            endCursor: z.string().nullable(),
          }),
        }),
      })
      .strict(),
  }),
});

export type PrListGraphQlOutput = z.infer<typeof prListGraphQlOutputSchema>;

export const prListOutputSchema = prListGraphQlOutputSchema;

export type PrListOutput = z.infer<typeof prListOutputSchema>;
