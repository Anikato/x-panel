import assert from 'node:assert/strict'
import test from 'node:test'
import { filterAddressIfaces, filterDNS, filterInventoryNics, filterTrafficNics, formatLinkSpeed, isNoiseIface } from './network.ts'

test('isNoiseIface drops docker and veth, keeps bridges and nics', () => {
  assert.equal(isNoiseIface('docker0'), true)
  assert.equal(isNoiseIface('veth1234'), true)
  assert.equal(isNoiseIface('br-ab12cd34ef56'), true)
  assert.equal(isNoiseIface('vmbr0'), false)
  assert.equal(isNoiseIface('nic1'), false)
  assert.equal(isNoiseIface('vmbr99'), false)
})

test('filterDNS drops link-local IPv6 with zone id', () => {
  assert.deepEqual(
    filterDNS(['fe80::4cbe:a209:33e5:f000%vmbr1', '1.1.1.1', '127.0.0.1']),
    ['1.1.1.1'],
  )
})

test('filterAddressIfaces keeps vmbr and drops docker0', () => {
  const list = filterAddressIfaces([
    { name: 'vmbr0', ipv4: ['192.168.1.2/24'], status: 'up' },
    { name: 'docker0', ipv4: ['172.17.0.1/16'], status: 'up' },
    { name: 'vmbr1', ipv4: ['10.10.10.2/24'], status: 'up' },
  ])
  assert.deepEqual(list.map((i) => i.name), ['vmbr0', 'vmbr1'])
})

test('filterInventoryNics keeps unplugged nics and drops docker/veth', () => {
  const list = filterInventoryNics([
    { name: 'eth0', ipv4: ['10.0.0.2/24'], status: 'up', connected: true, kind: 'ethernet', speedMbps: 1000 },
    { name: 'eth1', ipv4: [], status: 'up', connected: false, kind: 'ethernet', speedMbps: 0 },
    { name: 'wlan0', ipv4: [], status: 'down', connected: false, kind: 'wifi', speedMbps: 0 },
    { name: 'docker0', ipv4: ['172.17.0.1/16'], status: 'up', connected: true, kind: 'bridge', speedMbps: 10000 },
    { name: 'vethabcd', ipv4: [], status: 'up', connected: true, kind: 'virtual', speedMbps: 0 },
    { name: 'vmbr0', ipv4: ['192.168.1.2/24'], status: 'up', connected: true, kind: 'bridge', speedMbps: 10000 },
  ])
  assert.deepEqual(list.map((i) => i.name), ['eth0', 'eth1', 'wlan0', 'vmbr0'])
})

test('formatLinkSpeed renders negotiated rates and blanks unknown', () => {
  assert.equal(formatLinkSpeed(1000), '1 Gbps')
  assert.equal(formatLinkSpeed(2500), '2.5 Gbps')
  assert.equal(formatLinkSpeed(100), '100 Mbps')
  assert.equal(formatLinkSpeed(0), '')
  assert.equal(formatLinkSpeed(undefined), '')
})

test('filterTrafficNics hides idle nics when some have traffic', () => {
  const list = filterTrafficNics([
    { name: 'nic1', speedUp: 66500, speedDown: 33700 },
    { name: 'nic2', speedUp: 0, speedDown: 0 },
    { name: 'nic3', speedUp: 0, speedDown: 0 },
    { name: 'nic4', speedUp: 0, speedDown: 0 },
    { name: 'vmbr0', speedUp: 0, speedDown: 0 },
    { name: 'vmbr1', speedUp: 1500, speedDown: 380 },
    { name: 'docker0', speedUp: 800, speedDown: 800 },
  ])
  assert.deepEqual(list.map((n) => n.name), ['nic1', 'vmbr1'])
})
