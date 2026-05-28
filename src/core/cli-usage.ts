type ExitWithUsageOptions = {
  usage: string;
  message?: string;
  exitCode?: number;
};

export function exitWithUsage(options: ExitWithUsageOptions): never {
  const { usage, message, exitCode = 1 } = options;

  if (message) {
    console.error(`Error: ${message}\n`);
  }

  console.log(usage);
  process.exit(exitCode);
}
