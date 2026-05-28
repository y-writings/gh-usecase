import { z } from 'zod';

const MAX_GRAPHQL_PAGE_SIZE = 100;
const DEFAULT_GRAPHQL_PAGE_SIZE = 30;

export const ownerParamSchema = z.string().min(1, '--owner is required');

export const nameParamSchema = z.string().min(1, '--name is required');

export const pullRequestNumberParamSchema = z.coerce
  .number()
  .int()
  .positive('--number must be a positive integer');

export const stateParamSchema = z
  .enum(['OPEN', 'CLOSED', 'MERGED'], {
    errorMap: () => ({ message: '--state must be OPEN, CLOSED, or MERGED' }),
  })
  .optional();

export const afterParamSchema = z.string().optional();

export const firstParamSchema = z.coerce
  .number()
  .int()
  .positive()
  .max(MAX_GRAPHQL_PAGE_SIZE)
  .default(DEFAULT_GRAPHQL_PAGE_SIZE);
