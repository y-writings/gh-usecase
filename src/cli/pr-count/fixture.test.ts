import { describe, expect, it } from 'bun:test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { buildFilters } from './query';
import { prCountOutputSchema } from './schema';

function readJson(path: string): unknown {
  return JSON.parse(readFileSync(resolve(import.meta.dir, '../../../', path), 'utf8'));
}

describe('pr-count fixture parity', () => {
  it('keeps the current output contract', () => {
    const input = prCountOutputSchema.parse(readJson('testdata/pr-count/basic.graphql.json'));
    const expected = prCountOutputSchema.parse(readJson('testdata/pr-count/basic.expected.json'));

    expect(input).toEqual(expected);
  });

  it('builds current GraphQL filters from CLI args', () => {
    expect(buildFilters({ owner: 'acme', name: 'rocket', state: 'OPEN' })).toEqual([
      'owner=acme',
      'name=rocket',
      'state=OPEN',
    ]);
  });
});
