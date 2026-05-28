# CLI Design Standards

This document captures the standard design and implementation rules for this CLI codebase.
Use this as a guardrail for future refactors and new command implementations.

## Goals

- Keep command behavior predictable for CLI users.
- Keep responsibilities explicit and narrow across modules.
- Avoid hidden behavior in JavaScript post-processing.

## Standards

1. **Single-page response contract for list-style commands**
   - List-style commands should return one page per call.
   - Pagination should be user-driven with `after` + `first`.

2. **Server-side filtering only**
   - Only filters directly supported by GitHub GraphQL are allowed.
   - Do not add JavaScript-side filtering that changes result semantics.

3. **No hidden aggregation loops**
   - Do not loop through cursors internally to collect multiple pages.
   - If users need more pages, they call again with `endCursor`.

4. **Schema consistency across flow**
   - The response shape should be consistent from fetch to return.
   - Avoid intermediate output shapes that differ from the final contract.

5. **Clear parser boundaries**
   - `parseResponseTextWithSchema` handles `responseText -> JSON -> schema`.
   - Avoid mixed input contracts that mimic overloaded signatures.

6. **Minimal parameter surface**
   - Keep only meaningful, supported parameters.
   - Remove parameters that are not part of the intended list API contract.

7. **Single source of first truth**
   - `MAX_GRAPHQL_PAGE_SIZE` is defined in schema and reused in validation and runtime logic.

8. **Input parameter composition rule (CLI-wide)**
   - Shared input parameter schemas live in `src/cli/api/input-params-schema.ts`.
   - Define each parameter schema individually (`owner`, `name`, `state`, `after`, `first`).
   - Compose parameters in each command schema with `z.object({...})`.
   - Do not re-introduce grouped wrapper schemas like `repositoryParamsSchema`.

9. **State parameter contract**
   - `state` accepts only `OPEN | CLOSED | MERGED`.
   - `state` is optional for commands that include it.
   - If omitted, the GraphQL `state` filter is not sent (all states).

10. **Zod style guardrails**
    - Prefer straightforward Zod chains with built-in behavior over custom transforms.
    - Use `.default(...)` for defaulting instead of manual transform fallback logic.
    - Avoid `preprocess`, `pipe`, and `trim` unless there is a strict, demonstrated need.
    - Prefer Zod standard validation messages for `int`, `positive`, and `max` unless a custom message is required.

11. **Magic number policy for CLI params**
    - No magic numbers in schema chains.
    - Use named constants (for example `DEFAULT_GRAPHQL_PAGE_SIZE`) in shared schema definitions.

12. **GraphQL execution boundary**
    - Commands should call `runGraphQlCommandParsed(...)`.
    - Command modules must not combine `runGraphQlCommand(...)` and `parseResponseTextWithSchema(...)` directly.
    - Keep command code focused on query/filter assembly and output schema selection.

13. **Command registry single source rule (Zod)**
    - Define command entries in one `z.object({...})` only (for example command names and descriptions).
    - Derive command name validation with `.keyof()` from that object instead of maintaining a separate literal array/union.
    - Derive command usage/help output from the same object shape to avoid duplicated command lists.
    - Adding a new command must require editing only that one `z.object` definition.

## Current Contract Summary (Reference: PR-related commands)

- Inputs:
  - `owner`, `name`, `state? (OPEN|CLOSED|MERGED)`, `after?`, `first?`
- Output:
  - GitHub GraphQL pull request list page shape (`nodes` + `pageInfo`)

## Change Rule

When changing any CLI command, verify that:

- User-facing behavior remains explicit and unsurprising.
- Filtering and pagination behavior are visible in arguments and output.
- No hidden local filtering or hidden multi-page aggregation is introduced.
