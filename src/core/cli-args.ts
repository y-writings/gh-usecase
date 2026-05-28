export type ParsedCliInput = {
  options: Record<string, string>;
  positional: string[];
  helpRequested: boolean;
};

export function parseCliInput(argv: string[]): ParsedCliInput {
  const options: Record<string, string> = {};
  const positional: string[] = [];
  let helpRequested = false;

  for (let i = 0; i < argv.length; i += 1) {
    const token = argv[i];

    if (token === '--help' || token === '-h') {
      helpRequested = true;
      continue;
    }

    if (token.startsWith('--')) {
      const withoutPrefix = token.slice(2);

      if (!withoutPrefix) {
        continue;
      }

      const equalSignIndex = withoutPrefix.indexOf('=');
      if (equalSignIndex >= 0) {
        const key = withoutPrefix.slice(0, equalSignIndex);
        const value = withoutPrefix.slice(equalSignIndex + 1);
        if (key) {
          options[key] = value;
        }
        continue;
      }

      const nextToken = argv[i + 1];
      if (nextToken && !nextToken.startsWith('--')) {
        options[withoutPrefix] = nextToken;
        i += 1;
      }

      continue;
    }

    positional.push(token);
  }

  return {
    options,
    positional,
    helpRequested,
  };
}
