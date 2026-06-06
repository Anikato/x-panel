const INVALID_CWD_VALUES = new Set(['', 'undefined', 'null'])

export function normalizeTerminalCwd(cwd: unknown): string {
  if (typeof cwd !== 'string') return '/'

  const normalized = cwd.trim()
  if (INVALID_CWD_VALUES.has(normalized.toLowerCase())) return '/'

  return normalized.startsWith('/') ? normalized : `/${normalized}`
}

export function buildInitialCwdCommand(cwd: unknown): string {
  return `cd ${JSON.stringify(normalizeTerminalCwd(cwd))} && clear\n`
}
