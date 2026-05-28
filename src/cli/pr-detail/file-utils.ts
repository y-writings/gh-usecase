const LIKELY_GENERATED_PATH_PATTERNS: readonly RegExp[] = [
  /(^|\/)dist\//,
  /(^|\/)build\//,
  /(^|\/)coverage\//,
  /(^|\/)vendor\//,
  /(^|\/)generated\//,
  /(^|\/)__snapshots__\//,
  /\.min\.[a-z0-9]+$/i,
  /\.lock$/i,
  /^pnpm-lock\.yaml$/i,
  /^bun\.lockb$/i,
  /^yarn\.lock$/i,
  /^package-lock\.json$/i,
  /^Cargo\.lock$/,
];

const LIKELY_BINARY_EXTENSIONS = new Set<string>([
  'png',
  'jpg',
  'jpeg',
  'gif',
  'webp',
  'bmp',
  'ico',
  'svg',
  'pdf',
  'zip',
  'gz',
  'tar',
  'rar',
  '7z',
  'mp3',
  'mp4',
  'mov',
  'avi',
  'wav',
  'ogg',
  'ttf',
  'otf',
  'woff',
  'woff2',
  'eot',
  'jar',
  'exe',
  'dll',
  'so',
  'dylib',
  'class',
]);

export function extractFileExtension(path: string): string | null {
  const dotIndex = path.lastIndexOf('.');
  if (dotIndex < 0 || dotIndex === path.length - 1) {
    return null;
  }

  return path.slice(dotIndex + 1).toLowerCase();
}

export function matchesAnyPattern(path: string, patterns: readonly RegExp[]): boolean {
  return patterns.some((pattern) => pattern.test(path));
}

export function hasKnownExtension(path: string, extensions: ReadonlySet<string>): boolean {
  const extension = extractFileExtension(path);
  if (!extension) {
    return false;
  }

  return extensions.has(extension);
}

export function isLikelyGeneratedFile(path: string): boolean {
  return matchesAnyPattern(path, LIKELY_GENERATED_PATH_PATTERNS);
}

export function isLikelyBinaryFile(path: string): boolean {
  return hasKnownExtension(path, LIKELY_BINARY_EXTENSIONS);
}
