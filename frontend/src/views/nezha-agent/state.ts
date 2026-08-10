/**
 * Pure Nezha Agent management-page state model.
 * Mirrors backend dto.NezhaAgentStatus; no Vue/axios/runtime deps.
 */

/** Matches backend dto.NezhaAgentConflict. */
export interface NezhaAgentConflict {
  kind: string
  detail: string
  message: string
}

/**
 * Matches backend dto.NezhaAgentStatus.
 * Agent secret is never part of status — only secretConfigured is exposed.
 */
export interface NezhaAgentStatus {
  componentAvailable: boolean
  configured: boolean
  configHealthy: boolean
  configError: string
  active: boolean
  serviceState: string
  enabled: boolean
  desiredEnabled: boolean
  drift: boolean
  version: string
  uuid: string
  dashboardUrl: string
  server: string
  tls: boolean
  insecureTls: boolean
  secretConfigured: boolean
  remoteOperationsEnabled: boolean
  permissionsWarning: string
  serviceError: string
  conflicts: NezhaAgentConflict[]
}

/** Write-only configure form; clientSecret is never restored from status. */
export interface NezhaAgentForm {
  dashboardUrl: string
  clientSecret: string
  remoteOperationsEnabled: boolean
  enableAndStart: boolean
}

export type NezhaAgentViewKind =
  | 'component-missing'
  | 'pending'
  | 'corrupt'
  | 'conflict'
  | 'service-error'
  | 'running'
  | 'stopped'

export interface NezhaAgentView {
  kind: NezhaAgentViewKind
  canConfigure: boolean
  canStart: boolean
  canStop: boolean
  canRestart: boolean
  canEnable: boolean
  canDisable: boolean
  canViewLogs: boolean
  hasDrift: boolean
  statusLabelKey: string
}

/** Temporary runtime stop only — does not change boot enablement. */
export const temporaryStop = {
  operation: 'stop' as const,
  labelKey: 'nezhaAgent.action.stop',
}

/**
 * Complete disable (disable --now). Retains binary, config.yml, and UUID so the
 * node can re-enable without registering as a new Dashboard host.
 */
export const completeDisable = {
  operation: 'disable' as const,
  labelKey: 'nezhaAgent.action.completeDisable',
  retainNoteKey: 'nezhaAgent.action.completeDisableRetainNote',
}

const STATUS_LABEL_KEYS: Record<NezhaAgentViewKind, string> = {
  'component-missing': 'nezhaAgent.status.componentMissing',
  pending: 'nezhaAgent.status.pending',
  corrupt: 'nezhaAgent.status.corrupt',
  conflict: 'nezhaAgent.status.conflict',
  'service-error': 'nezhaAgent.status.serviceError',
  running: 'nezhaAgent.status.running',
  stopped: 'nezhaAgent.status.stopped',
}

/**
 * Build a fresh configure form.
 * clientSecret is always a brand-new empty string and is never read from status.
 * First-time pending defaults: remote ops on + enableAndStart on.
 * Healthy existing config: reuse remoteOperationsEnabled, enableAndStart off.
 */
export function createNezhaAgentForm(status: NezhaAgentStatus): NezhaAgentForm {
  const clientSecret = ''
  if (status.configHealthy) {
    return {
      dashboardUrl: status.dashboardUrl ?? '',
      clientSecret,
      remoteOperationsEnabled: status.remoteOperationsEnabled,
      enableAndStart: false,
    }
  }
  // Pending / missing / corrupt: never restore secret; first-time pending enables start.
  if (!status.configured) {
    return {
      dashboardUrl: status.dashboardUrl ?? '',
      clientSecret,
      remoteOperationsEnabled: true,
      enableAndStart: true,
    }
  }
  return {
    dashboardUrl: status.dashboardUrl ?? '',
    clientSecret,
    remoteOperationsEnabled: status.remoteOperationsEnabled,
    enableAndStart: false,
  }
}

/**
 * Derive UI kind and action gates from status.
 *
 * Priority (highest first):
 * 1. component-missing — no bundled binary
 * 2. pending — config file absent
 * 3. corrupt — config present but unhealthy (no DB overwrite path)
 * 4. conflict — external unit/process/directory blocks start/restart/enable
 * 5. service-error — serviceError set or systemd state failed
 * 6. running — active
 * 7. stopped — healthy configured, not active
 */
/** Safe-exit gates shared by degraded kinds that must not launch. */
function safeExitGates(status: NezhaAgentStatus) {
  return {
    canStart: false,
    canRestart: false,
    canEnable: false,
    canStop: status.active,
    canDisable: status.active || status.enabled || status.desiredEnabled,
    canViewLogs: true,
  }
}

export function deriveNezhaAgentView(status: NezhaAgentStatus): NezhaAgentView {
  const kind = deriveKind(status)
  const hasDrift = status.drift
  const statusLabelKey = STATUS_LABEL_KEYS[kind]

  switch (kind) {
    case 'component-missing':
      return {
        kind,
        canConfigure: false,
        ...safeExitGates(status),
        hasDrift,
        statusLabelKey,
      }
    case 'pending':
      return {
        kind,
        canConfigure: true,
        ...safeExitGates(status),
        hasDrift,
        statusLabelKey,
      }
    case 'corrupt':
      return {
        kind,
        canConfigure: false,
        ...safeExitGates(status),
        hasDrift,
        statusLabelKey,
      }
    case 'conflict':
      return {
        kind,
        canConfigure: true,
        canStart: false,
        canRestart: false,
        canEnable: false,
        canStop: status.active,
        canDisable: true,
        canViewLogs: true,
        hasDrift,
        statusLabelKey,
      }
    case 'service-error':
      // Fail closed on launch actions: ServiceError may include conflict-probe failures.
      return {
        kind,
        canConfigure: true,
        ...safeExitGates(status),
        hasDrift,
        statusLabelKey,
      }
    case 'running':
      return {
        kind,
        canConfigure: true,
        canStart: false,
        canStop: true,
        canRestart: true,
        canEnable: !status.enabled,
        // Same safe-exit rule as degraded kinds: cover live process, unit enable, or DB desired.
        canDisable: status.active || status.enabled || status.desiredEnabled,
        canViewLogs: true,
        hasDrift,
        statusLabelKey,
      }
    case 'stopped':
      return {
        kind,
        canConfigure: true,
        canStart: true,
        canStop: false,
        canRestart: false,
        canEnable: !status.enabled,
        // Same safe-exit rule as degraded kinds: cover live process, unit enable, or DB desired.
        canDisable: status.active || status.enabled || status.desiredEnabled,
        canViewLogs: true,
        hasDrift,
        statusLabelKey,
      }
  }
}

function deriveKind(status: NezhaAgentStatus): NezhaAgentViewKind {
  if (!status.componentAvailable) {
    return 'component-missing'
  }
  if (!status.configured) {
    return 'pending'
  }
  if (!status.configHealthy) {
    return 'corrupt'
  }
  if (Array.isArray(status.conflicts) && status.conflicts.length > 0) {
    return 'conflict'
  }
  if (
    (typeof status.serviceError === 'string' && status.serviceError.trim() !== '') ||
    status.serviceState === 'failed'
  ) {
    return 'service-error'
  }
  if (status.active) {
    return 'running'
  }
  return 'stopped'
}
