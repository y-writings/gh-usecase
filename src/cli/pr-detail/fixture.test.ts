import { describe, expect, it } from 'bun:test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { prDetailGraphQlOutputSchema, prDetailOutputSchema } from './schema';
import { transformOutput } from './transform';

function readJson(path: string): unknown {
  return JSON.parse(readFileSync(resolve(import.meta.dir, '../../../', path), 'utf8'));
}

describe('pr-detail fixture parity', () => {
  it('transforms GraphQL response to the current output contract', () => {
    const input = prDetailGraphQlOutputSchema.parse(
      readJson('testdata/pr-detail/basic.graphql.json'),
    );
    const expected = prDetailOutputSchema.parse(readJson('testdata/pr-detail/basic.expected.json'));

    expect(transformOutput(input)).toEqual(expected);
  });
});
