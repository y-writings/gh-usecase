import type { ZodTypeAny } from 'zod';

export class ResponseParseError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'ResponseParseError';
  }
}

export function parseResponseJson(responseText: string): unknown {
  try {
    return JSON.parse(responseText);
  } catch {
    throw new ResponseParseError('failed to parse response as JSON');
  }
}

type ParseResponseWithSchemaInput<TSchema extends ZodTypeAny> = {
  parsedJson: unknown;
  outputSchema: TSchema;
};

type ParseResponseTextWithSchemaInput<TSchema extends ZodTypeAny> = {
  responseText: string;
  outputSchema: TSchema;
};

export function parseResponseWithSchema<TSchema extends ZodTypeAny>({
  parsedJson,
  outputSchema,
}: ParseResponseWithSchemaInput<TSchema>): TSchema['_output'] {
  const parsedOutput = outputSchema.safeParse(parsedJson);
  if (!parsedOutput.success) {
    throw new ResponseParseError('unexpected response format');
  }

  return parsedOutput.data;
}

export function parseResponseTextWithSchema<TSchema extends ZodTypeAny>({
  responseText,
  outputSchema,
}: ParseResponseTextWithSchemaInput<TSchema>): TSchema['_output'] {
  return parseResponseWithSchema({
    parsedJson: parseResponseJson(responseText),
    outputSchema,
  });
}
