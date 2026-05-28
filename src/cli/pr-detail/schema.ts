import { z } from 'zod';
import {
  nameParamSchema,
  ownerParamSchema,
  pullRequestNumberParamSchema,
} from '../api/input-params-schema';

const MAX_FILE_PAGE_SIZE = 100;
const DEFAULT_FILE_PAGE_SIZE = 40;

export const filesFirstParamSchema = z.coerce
  .number()
  .int()
  .positive()
  .max(MAX_FILE_PAGE_SIZE)
  .default(DEFAULT_FILE_PAGE_SIZE);

export const cliArgsSchema = z.object({
  owner: ownerParamSchema,
  name: nameParamSchema,
  number: pullRequestNumberParamSchema,
  filesFirst: filesFirstParamSchema,
});

export type CliArgs = z.infer<typeof cliArgsSchema>;

const graphqlActorSchema = z
  .object({
    login: z.string(),
  })
  .nullable();

const graphqlReviewNodeSchema = z.object({
  id: z.string(),
  author: graphqlActorSchema,
  state: z.enum(['APPROVED', 'CHANGES_REQUESTED', 'COMMENTED', 'DISMISSED', 'PENDING']),
  bodyText: z.string(),
  submittedAt: z.string().nullable(),
  commit: z
    .object({
      oid: z.string(),
    })
    .nullable(),
});

const graphqlReviewCommentNodeSchema = z.object({
  id: z.string(),
  author: graphqlActorSchema,
  bodyText: z.string(),
  path: z.string().nullable(),
  createdAt: z.string(),
  line: z.number().int().nonnegative().nullable(),
  originalLine: z.number().int().nonnegative().nullable(),
  startLine: z.number().int().nonnegative().nullable(),
  originalStartLine: z.number().int().nonnegative().nullable(),
  side: z.enum(['LEFT', 'RIGHT']).nullable(),
  startSide: z.enum(['LEFT', 'RIGHT']).nullable(),
  commit: z
    .object({
      oid: z.string(),
    })
    .nullable(),
  originalCommit: z
    .object({
      oid: z.string(),
    })
    .nullable(),
});

const graphqlReviewThreadSchema = z.object({
  isResolved: z.boolean(),
  comments: z.object({
    nodes: z.array(graphqlReviewCommentNodeSchema),
  }),
});

const graphqlFileNodeSchema = z.object({
  path: z.string(),
  additions: z.number().int().nonnegative(),
  deletions: z.number().int().nonnegative(),
  changeType: z.enum(['ADDED', 'COPIED', 'DELETED', 'MODIFIED', 'RENAMED', 'CHANGED']),
});

const graphqlCommitNodeSchema = z.object({
  commit: z.object({
    oid: z.string(),
    messageHeadline: z.string(),
    committedDate: z.string(),
  }),
});

const graphqlPullRequestSchema = z.object({
  number: z.number().int().positive(),
  title: z.string(),
  bodyText: z.string(),
  reviewDecision: z.enum(['APPROVED', 'CHANGES_REQUESTED', 'REVIEW_REQUIRED']).nullable(),
  author: graphqlActorSchema,
  mergeCommit: z
    .object({
      oid: z.string(),
    })
    .nullable(),
  baseRefOid: z.string(),
  headRefOid: z.string(),
  additions: z.number().int().nonnegative(),
  deletions: z.number().int().nonnegative(),
  changedFiles: z.number().int().nonnegative(),
  reviews: z.object({
    nodes: z.array(graphqlReviewNodeSchema),
  }),
  reviewThreads: z.object({
    nodes: z.array(graphqlReviewThreadSchema),
  }),
  files: z.object({
    totalCount: z.number().int().nonnegative(),
    nodes: z.array(graphqlFileNodeSchema),
    pageInfo: z.object({
      hasNextPage: z.boolean(),
      endCursor: z.string().nullable(),
    }),
  }),
  commits: z.object({
    nodes: z.array(graphqlCommitNodeSchema),
  }),
});

export const prDetailGraphQlOutputSchema = z.object({
  data: z.object({
    repository: z
      .object({
        pullRequest: graphqlPullRequestSchema.nullable(),
      })
      .strict(),
  }),
});

export type PrDetailGraphQlOutput = z.infer<typeof prDetailGraphQlOutputSchema>;

export const prDetailOutputSchema = z.object({
  data: z.object({
    repository: z
      .object({
        pullRequest: z
          .object({
            number: z.number().int().positive(),
            title: z.string(),
            description: z.string(),
            reviewDecision: z.enum(['APPROVED', 'CHANGES_REQUESTED', 'REVIEW_REQUIRED']).nullable(),
            authorLogin: z.string().nullable(),
            mergeCommitOid: z.string().nullable(),
            reviewStartCommitOid: z.string().nullable(),
            reviewStartConfidence: z.enum(['high', 'medium', 'low']),
            codeDiff: z.object({
              stats: z.object({
                changedFiles: z.number().int().nonnegative(),
                additions: z.number().int().nonnegative(),
                deletions: z.number().int().nonnegative(),
              }),
              files: z.array(
                z.object({
                  path: z.string(),
                  changeType: z.enum([
                    'ADDED',
                    'COPIED',
                    'DELETED',
                    'MODIFIED',
                    'RENAMED',
                    'CHANGED',
                  ]),
                  additions: z.number().int().nonnegative(),
                  deletions: z.number().int().nonnegative(),
                }),
              ),
              excludedFiles: z.array(
                z.object({
                  path: z.string(),
                  reason: z.enum(['likely-binary', 'likely-generated']),
                }),
              ),
              filePageInfo: z.object({
                hasNextPage: z.boolean(),
                endCursor: z.string().nullable(),
                totalCount: z.number().int().nonnegative(),
              }),
              strategy: z.object({
                baseCommit: z.string(),
                headCommit: z.string(),
              }),
            }),
            conversations: z.object({
              reviews: z.array(
                z.object({
                  authorLogin: z.string().nullable(),
                  state: z.enum([
                    'APPROVED',
                    'CHANGES_REQUESTED',
                    'COMMENTED',
                    'DISMISSED',
                    'PENDING',
                  ]),
                  body: z.string(),
                  submittedAt: z.string().nullable(),
                  commitOid: z.string().nullable(),
                }),
              ),
              reviewThreads: z.array(
                z.object({
                  isResolved: z.boolean(),
                  comments: z.array(
                    z.object({
                      id: z.string(),
                      authorLogin: z.string().nullable(),
                      body: z.string(),
                      path: z.string().nullable(),
                      createdAt: z.string(),
                      line: z.number().int().nonnegative().nullable(),
                      originalLine: z.number().int().nonnegative().nullable(),
                      startLine: z.number().int().nonnegative().nullable(),
                      originalStartLine: z.number().int().nonnegative().nullable(),
                      side: z.enum(['LEFT', 'RIGHT']).nullable(),
                      startSide: z.enum(['LEFT', 'RIGHT']).nullable(),
                      commitOid: z.string().nullable(),
                      originalCommitOid: z.string().nullable(),
                    }),
                  ),
                }),
              ),
            }),
            commits: z.array(
              z.object({
                oid: z.string(),
                messageHeadline: z.string(),
                committedDate: z.string(),
              }),
            ),
          })
          .nullable(),
      })
      .strict(),
  }),
});

export type PrDetailOutput = z.infer<typeof prDetailOutputSchema>;
