import assert from 'node:assert/strict'
import test from 'node:test'
import { buildUnitPreview, serviceUnitName } from './unit-preview.ts'

test('serviceUnitName prefixes xp- on create', () => {
  assert.equal(serviceUnitName('app', false), 'xp-app')
  assert.equal(serviceUnitName('xp-app', false), 'xp-app')
  assert.equal(serviceUnitName('nginx', true), 'nginx')
  assert.equal(serviceUnitName('  ', false), '')
})

test('buildUnitPreview matches systemd unit shape and defaults', () => {
  const text = buildUnitPreview({
    name: 'xp-app',
    execStart: '/usr/bin/python3 /opt/app/main.py',
    restartSec: 5,
  })
  assert.equal(
    text,
    [
      '[Unit]',
      'Description=xp-app managed by X-Panel',
      'After=network.target',
      '',
      '[Service]',
      'Type=simple',
      'ExecStart=/usr/bin/python3 /opt/app/main.py',
      'Restart=on-failure',
      'RestartSec=5',
      'StandardOutput=journal',
      'StandardError=journal',
      '',
      '[Install]',
      'WantedBy=multi-user.target',
      '',
    ].join('\n'),
  )
})

test('buildUnitPreview writes optional fields and one Environment per line', () => {
  const text = buildUnitPreview({
    name: 'xp-app',
    description: 'demo',
    execStart: '/opt/app/bin',
    execStartPre: '/bin/true',
    workingDir: '/opt/app',
    user: 'www-data',
    environment: 'NODE_ENV=production\n\nPORT=3000',
    restart: 'always',
    restartSec: 0,
    afterTarget: 'docker.service',
  })
  assert.match(text, /Description=demo/)
  assert.match(text, /After=docker.service/)
  assert.match(text, /ExecStartPre=\/bin\/true/)
  assert.match(text, /WorkingDirectory=\/opt\/app/)
  assert.match(text, /User=www-data/)
  assert.match(text, /Environment=NODE_ENV=production/)
  assert.match(text, /Environment=PORT=3000/)
  assert.doesNotMatch(text, /RestartSec=/)
  assert.equal((text.match(/^Environment=/gm) || []).length, 2)
})
