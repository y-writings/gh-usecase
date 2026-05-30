import { describe, expect, it } from 'bun:test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { buildFilters } from './query';
import { prListOutputSchema } from './schema';

function readJson(path: string): unknown {
  return JSON.parse(readFileSync(resolve(import.meta.dir, '../../../', path), 'utf8'));
}

describe('pr-list fixture parity', () => {
  it('keeps the current output contract', () => {
    const input = prListOutputSchema.parse(readJson('testdata/pr-list/basic.graphql.json'));
    const expected = prListOutputSchema.parse(readJson('testdata/pr-list/basic.expected.json'));

    expect(input).toEqual(expected);
  });

  it('builds current GraphQL filters from CLI args', () => {
    expect(
      buildFilters({
        owner: 'acme',
        name: 'rocket',
        state: 'OPEN',
        after: 'cursor-1',
        first: 25,
      }),
    ).toEqual(['owner=acme', 'name=rocket', 'state=OPEN', 'after=cursor-1', 'first=25']);
  });
});
