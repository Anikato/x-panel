import assert from 'node:assert/strict'
import test from 'node:test'
import {
  completeDisable,
  createNezhaAgentForm,
  deriveNezhaAgentView,
  temporaryStop,
  type NezhaAgentStatus,
} from './state.ts'

function baseStatus(overrides: Partial<NezhaAgentStatus> = {}): NezhaAgentStatus {
  return {
    componentAvailable: true,
    configured: true,
    configHealthy: true,
    configError: '',
    active: false,
    serviceState: 'inactive',
    enabled: false,
    desiredEnabled: false,
    drift: false,
    version: '2.3.1',
    uuid: 'uuid-test',
    dashboardUrl: 'https://dash.example.com',
    server: 'dash.example.com:443',
    tls: true,
    insecureTls: false,
    secretConfigured: true,
    remoteOperationsEnabled: true,
    permissionsWarning: '',
    serviceError: '',
    conflicts: [],
    ...overrides,
  }
}

test('pending: missing config is configurable but not startable', () => {
  const status = baseStatus({
    configured: false,
    configHealthy: false,
    configError: 'nezha agent config missing',
    secretConfigured: false,
    dashboardUrl: '',
    server: '',
    uuid: '',
    remoteOperationsEnabled: false,
  })
  const vm = deriveNezhaAgentView(status)
  assert.equal(vm.kind, 'pending')
  assert.equal(vm.canConfigure, true)
  assert.equal(vm.canStart, false)
  assert.equal(vm.canStop, false)
  assert.equal(vm.canRestart, false)
  assert.equal(vm.canEnable, false)
  assert.equal(vm.canDisable, false)
  assert.equal(typeof vm.statusLabelKey, 'string')
  assert.ok(vm.statusLabelKey.length > 0)

  const form = createNezhaAgentForm(status)
  assert.equal(form.clientSecret, '')
  assert.equal(form.remoteOperationsEnabled, true)
  assert.equal(form.enableAndStart, true)
})

test('stopped: healthy inactive agent can start; enabled and active stay separate', () => {
  const status = baseStatus({
    active: false,
    serviceState: 'inactive',
    enabled: true,
    desiredEnabled: true,
  })
  const vm = deriveNezhaAgentView(status)
  assert.equal(vm.kind, 'stopped')
  assert.equal(vm.canConfigure, true)
  assert.equal(vm.canStart, true)
  assert.equal(vm.canStop, false)
  assert.equal(vm.canRestart, false)
  assert.equal(vm.canEnable, false)
  assert.equal(vm.canDisable, true)
  assert.equal(vm.canViewLogs, true)
})

test('running: can stop and restart, cannot start', () => {
  const status = baseStatus({
    active: true,
    serviceState: 'active',
    enabled: true,
    desiredEnabled: true,
  })
  const vm = deriveNezhaAgentView(status)
  assert.equal(vm.kind, 'running')
  assert.equal(vm.canStart, false)
  assert.equal(vm.canStop, true)
  assert.equal(vm.canRestart, true)
  assert.equal(vm.canEnable, false)
  assert.equal(vm.canDisable, true)
  assert.equal(vm.canConfigure, true)
  assert.equal(vm.canViewLogs, true)
})

// Regression: active but not boot-enabled must still offer complete disable
// so disable --now can stop the live process and clear DB desired.
test('running: active=true enabled=false still allows complete disable', () => {
  const vm = deriveNezhaAgentView(
    baseStatus({
      active: true,
      serviceState: 'active',
      enabled: false,
      desiredEnabled: false,
      drift: false,
    }),
  )
  assert.equal(vm.kind, 'running')
  assert.equal(vm.canStop, true)
  assert.equal(vm.canRestart, true)
  assert.equal(vm.canEnable, true)
  assert.equal(vm.canDisable, true)
  assert.equal(vm.canStart, false)
})

// Regression: stopped with desiredEnabled drift must still allow complete disable
// so disable --now can set DB desired to false even when unit is already inactive.
test('stopped: active=false enabled=false desiredEnabled=true still allows complete disable', () => {
  const vm = deriveNezhaAgentView(
    baseStatus({
      active: false,
      serviceState: 'inactive',
      enabled: false,
      desiredEnabled: true,
      drift: true,
    }),
  )
  assert.equal(vm.kind, 'stopped')
  assert.equal(vm.canStart, true)
  assert.equal(vm.canStop, false)
  assert.equal(vm.canRestart, false)
  assert.equal(vm.canEnable, true)
  assert.equal(vm.canDisable, true)
  assert.equal(vm.hasDrift, true)
})

test('corrupt: cannot start and must not offer DB overwrite path', () => {
  const status = baseStatus({
    configured: true,
    configHealthy: false,
    configError: 'nezha agent config is not valid YAML',
    active: false,
    secretConfigured: false,
  })
  const vm = deriveNezhaAgentView(status)
  assert.equal(vm.kind, 'corrupt')
  assert.equal(vm.canStart, false)
  assert.equal(vm.canRestart, false)
  assert.equal(vm.canEnable, false)
  // No DB rebuild/overwrite path: pure model refuses configure-from-status recovery.
  assert.equal(vm.canConfigure, false)
  assert.equal(vm.canViewLogs, true)
})

test('external conflict blocks start/restart/enable but allows stop/disable', () => {
  const stopped = deriveNezhaAgentView(
    baseStatus({
      active: false,
      enabled: true,
      conflicts: [
        {
          kind: 'unit',
          detail: 'nezha-agent.service',
          message: 'external unit',
        },
      ],
    }),
  )
  assert.equal(stopped.kind, 'conflict')
  assert.equal(stopped.canStart, false)
  assert.equal(stopped.canRestart, false)
  assert.equal(stopped.canEnable, false)
  assert.equal(stopped.canStop, false)
  assert.equal(stopped.canDisable, true)

  const running = deriveNezhaAgentView(
    baseStatus({
      active: true,
      serviceState: 'active',
      enabled: true,
      conflicts: [
        {
          kind: 'process',
          detail: '/opt/nezha/agent/nezha-agent',
          message: 'external process',
        },
      ],
    }),
  )
  assert.equal(running.kind, 'conflict')
  assert.equal(running.canStart, false)
  assert.equal(running.canRestart, false)
  assert.equal(running.canEnable, false)
  assert.equal(running.canStop, true)
  assert.equal(running.canDisable, true)
  assert.equal(running.canViewLogs, true)
})

test('service failure is service-error and keeps log entry', () => {
  const byError = deriveNezhaAgentView(
    baseStatus({
      active: false,
      serviceState: 'inactive',
      serviceError: 'systemctl show failed',
    }),
  )
  assert.equal(byError.kind, 'service-error')
  assert.equal(byError.canViewLogs, true)

  const byFailed = deriveNezhaAgentView(
    baseStatus({
      active: false,
      serviceState: 'failed',
      serviceError: '',
    }),
  )
  assert.equal(byFailed.kind, 'service-error')
  assert.equal(byFailed.canViewLogs, true)
})

test('service-error fail-closed: start/restart/enable false; stop/disable when active/enabled', () => {
  // ServiceError may come from conflict-detection failure; backend requires fail closed
  // on all launch-class actions while still allowing safe exit + logs.
  const activeEnabled = deriveNezhaAgentView(
    baseStatus({
      active: true,
      serviceState: 'active',
      enabled: true,
      desiredEnabled: true,
      serviceError: 'conflict probe failed',
    }),
  )
  assert.equal(activeEnabled.kind, 'service-error')
  assert.equal(activeEnabled.canStart, false)
  assert.equal(activeEnabled.canRestart, false)
  assert.equal(activeEnabled.canEnable, false)
  assert.equal(activeEnabled.canStop, true)
  assert.equal(activeEnabled.canDisable, true)
  assert.equal(activeEnabled.canViewLogs, true)
  assert.equal(activeEnabled.canConfigure, true)

  const inactive = deriveNezhaAgentView(
    baseStatus({
      active: false,
      serviceState: 'failed',
      enabled: false,
      desiredEnabled: false,
      serviceError: 'systemctl show failed',
    }),
  )
  assert.equal(inactive.kind, 'service-error')
  assert.equal(inactive.canStart, false)
  assert.equal(inactive.canRestart, false)
  assert.equal(inactive.canEnable, false)
  assert.equal(inactive.canStop, false)
  assert.equal(inactive.canDisable, false)
  assert.equal(inactive.canViewLogs, true)
})

test('corrupt with active/enabled allows safe stop/disable; launch actions stay false', () => {
  const vm = deriveNezhaAgentView(
    baseStatus({
      configured: true,
      configHealthy: false,
      configError: 'nezha agent config is not valid YAML',
      active: true,
      serviceState: 'active',
      enabled: true,
      desiredEnabled: true,
      secretConfigured: false,
    }),
  )
  assert.equal(vm.kind, 'corrupt')
  assert.equal(vm.canConfigure, false)
  assert.equal(vm.canStart, false)
  assert.equal(vm.canRestart, false)
  assert.equal(vm.canEnable, false)
  assert.equal(vm.canStop, true)
  assert.equal(vm.canDisable, true)
  assert.equal(vm.canViewLogs, true)
})

test('component-missing with enabled allows safe disable and logs; no launch/configure', () => {
  const vm = deriveNezhaAgentView(
    baseStatus({
      componentAvailable: false,
      configured: true,
      configHealthy: true,
      active: false,
      enabled: true,
      desiredEnabled: true,
    }),
  )
  assert.equal(vm.kind, 'component-missing')
  assert.equal(vm.canConfigure, false)
  assert.equal(vm.canStart, false)
  assert.equal(vm.canRestart, false)
  assert.equal(vm.canEnable, false)
  assert.equal(vm.canStop, false)
  assert.equal(vm.canDisable, true)
  assert.equal(vm.canViewLogs, true)
})

test('pending with active allows safe stop/disable and logs; no launch actions', () => {
  const vm = deriveNezhaAgentView(
    baseStatus({
      configured: false,
      configHealthy: false,
      configError: 'nezha agent config missing',
      active: true,
      serviceState: 'active',
      enabled: true,
      desiredEnabled: true,
      secretConfigured: false,
      dashboardUrl: '',
      server: '',
      uuid: '',
    }),
  )
  assert.equal(vm.kind, 'pending')
  assert.equal(vm.canConfigure, true)
  assert.equal(vm.canStart, false)
  assert.equal(vm.canRestart, false)
  assert.equal(vm.canEnable, false)
  assert.equal(vm.canStop, true)
  assert.equal(vm.canDisable, true)
  assert.equal(vm.canViewLogs, true)
})

test('drift is surfaced as-is on the view model', () => {
  const drifted = deriveNezhaAgentView(
    baseStatus({
      enabled: true,
      desiredEnabled: false,
      drift: true,
    }),
  )
  assert.equal(drifted.hasDrift, true)

  const clean = deriveNezhaAgentView(
    baseStatus({
      enabled: true,
      desiredEnabled: true,
      drift: false,
    }),
  )
  assert.equal(clean.hasDrift, false)
})

test('createNezhaAgentForm always blanks secret; healthy config reuses remote ops defaults', () => {
  const healthy = baseStatus({
    configHealthy: true,
    dashboardUrl: 'https://dash.example.com',
    remoteOperationsEnabled: false,
  })
  const form = createNezhaAgentForm(healthy)
  assert.equal(form.dashboardUrl, 'https://dash.example.com')
  assert.equal(form.clientSecret, '')
  assert.equal(form.remoteOperationsEnabled, false)
  assert.equal(form.enableAndStart, false)

  // Even if a polluted status object carries a secret field, never restore it.
  const polluted = {
    ...healthy,
    clientSecret: 'must-never-restore',
  } as NezhaAgentStatus & { clientSecret: string }
  assert.equal(createNezhaAgentForm(polluted).clientSecret, '')
})

test('two consecutive createNezhaAgentForm calls keep secret blank and return distinct objects', () => {
  const status = baseStatus({ configHealthy: true })
  const first = createNezhaAgentForm(status)
  const second = createNezhaAgentForm(status)
  assert.equal(first.clientSecret, '')
  assert.equal(second.clientSecret, '')
  assert.notEqual(first, second)
  first.clientSecret = 'mutated'
  assert.equal(second.clientSecret, '')
})

test('temporary stop and complete disable are distinct operations', () => {
  assert.equal(temporaryStop.operation, 'stop')
  assert.ok(temporaryStop.labelKey.includes('stop'))
  assert.ok(!temporaryStop.labelKey.toLowerCase().includes('disable'))

  assert.equal(completeDisable.operation, 'disable')
  assert.ok(completeDisable.labelKey.toLowerCase().includes('completedisable') || completeDisable.labelKey.includes('completeDisable'))
  assert.ok(typeof completeDisable.retainNoteKey === 'string')
  assert.ok(completeDisable.retainNoteKey.length > 0)
  assert.notEqual(temporaryStop.operation, completeDisable.operation)
  assert.notEqual(temporaryStop.labelKey, completeDisable.labelKey)
})

test('component missing cannot configure or start', () => {
  const vm = deriveNezhaAgentView(
    baseStatus({
      componentAvailable: false,
      configured: false,
      configHealthy: false,
    }),
  )
  assert.equal(vm.kind, 'component-missing')
  assert.equal(vm.canConfigure, false)
  assert.equal(vm.canStart, false)
  assert.equal(vm.canStop, false)
  assert.equal(vm.canRestart, false)
  assert.equal(vm.canEnable, false)
  assert.equal(vm.canDisable, false)
})

test('kind priority: component-missing > pending > corrupt > conflict > service-error > running/stopped', () => {
  assert.equal(
    deriveNezhaAgentView(
      baseStatus({
        componentAvailable: false,
        configured: false,
        configHealthy: false,
        conflicts: [{ kind: 'unit', detail: 'x', message: 'y' }],
        serviceError: 'err',
        active: true,
      }),
    ).kind,
    'component-missing',
  )
  assert.equal(
    deriveNezhaAgentView(
      baseStatus({
        configured: false,
        configHealthy: false,
        conflicts: [{ kind: 'unit', detail: 'x', message: 'y' }],
        serviceError: 'err',
      }),
    ).kind,
    'pending',
  )
  assert.equal(
    deriveNezhaAgentView(
      baseStatus({
        configured: true,
        configHealthy: false,
        conflicts: [{ kind: 'unit', detail: 'x', message: 'y' }],
        serviceError: 'err',
      }),
    ).kind,
    'corrupt',
  )
  assert.equal(
    deriveNezhaAgentView(
      baseStatus({
        conflicts: [{ kind: 'unit', detail: 'x', message: 'y' }],
        serviceError: 'err',
        active: true,
        serviceState: 'active',
      }),
    ).kind,
    'conflict',
  )
  assert.equal(
    deriveNezhaAgentView(
      baseStatus({
        serviceError: 'err',
        active: true,
        serviceState: 'active',
      }),
    ).kind,
    'service-error',
  )
  assert.equal(
    deriveNezhaAgentView(
      baseStatus({
        active: true,
        serviceState: 'active',
      }),
    ).kind,
    'running',
  )
  assert.equal(deriveNezhaAgentView(baseStatus({ active: false })).kind, 'stopped')
})

test('NezhaAgentStatus shape has no clientSecret field in the exported contract', () => {
  const status: NezhaAgentStatus = baseStatus()
  assert.equal('clientSecret' in status, false)
  // Compile-time contract is mirrored by runtime keys used by the panel.
  const keys = Object.keys(status)
  assert.ok(keys.includes('componentAvailable'))
  assert.ok(keys.includes('secretConfigured'))
  assert.ok(keys.includes('remoteOperationsEnabled'))
  assert.ok(keys.includes('conflicts'))
  assert.ok(!keys.includes('clientSecret'))
})
