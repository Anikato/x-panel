export interface UnitPreviewInput {
  name?: string
  description?: string
  type?: string
  execStart?: string
  execStartPre?: string
  execStopPost?: string
  workingDir?: string
  user?: string
  restart?: string
  restartSec?: number
  environment?: string
  afterTarget?: string
  stdOutput?: string
  stdError?: string
  timeoutStart?: number
  timeoutStop?: number
}

export function serviceUnitName(name: string, isEdit: boolean): string {
  const n = (name || '').trim()
  if (!n) return ''
  if (isEdit || n.startsWith('xp-')) return n
  return `xp-${n}`
}

export function buildUnitPreview(p: UnitPreviewInput): string {
  const name = (p.name || '').trim()
  const description = (p.description || '').trim() || (name ? `${name} managed by X-Panel` : '')
  const restart = p.restart || 'on-failure'
  const afterTarget = p.afterTarget || 'network.target'
  const type = p.type || 'simple'
  const stdOutput = p.stdOutput || 'journal'
  const stdError = p.stdError || 'journal'
  const execStart = (p.execStart || '').trim()
  const lines: string[] = [
    '[Unit]',
    `Description=${description}`,
    `After=${afterTarget}`,
    '',
    '[Service]',
    `Type=${type}`,
  ]
  if ((p.execStartPre || '').trim()) lines.push(`ExecStartPre=${p.execStartPre!.trim()}`)
  lines.push(`ExecStart=${execStart}`)
  if ((p.execStopPost || '').trim()) lines.push(`ExecStopPost=${p.execStopPost!.trim()}`)
  if ((p.workingDir || '').trim()) lines.push(`WorkingDirectory=${p.workingDir!.trim()}`)
  if ((p.user || '').trim()) lines.push(`User=${p.user!.trim()}`)
  for (const env of (p.environment || '').split('\n')) {
    const item = env.trim()
    if (item) lines.push(`Environment=${item}`)
  }
  lines.push(`Restart=${restart}`)
  if ((p.restartSec || 0) > 0) lines.push(`RestartSec=${p.restartSec}`)
  if ((p.timeoutStart || 0) > 0) lines.push(`TimeoutStartSec=${p.timeoutStart}`)
  if ((p.timeoutStop || 0) > 0) lines.push(`TimeoutStopSec=${p.timeoutStop}`)
  lines.push(`StandardOutput=${stdOutput}`)
  lines.push(`StandardError=${stdError}`)
  lines.push('')
  lines.push('[Install]')
  lines.push('WantedBy=multi-user.target')
  lines.push('')
  return lines.join('\n')
}
