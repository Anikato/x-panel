import assert from 'node:assert/strict'
import test from 'node:test'
import { classifySensor, fallbackSensorName, groupSensors } from './sensors.ts'

test('classifySensor maps common hwmon keys', () => {
  assert.equal(classifySensor('acpitz'), 'board')
  assert.equal(classifySensor('coretemp_package_id_0'), 'cpu')
  assert.equal(classifySensor('nvme_composite'), 'nvme')
  assert.equal(classifySensor('nvme_sensor_1'), 'nvme')
  assert.equal(classifySensor('k10temp_tctl'), 'cpu')
  assert.equal(classifySensor('amdgpu'), 'gpu')
  assert.equal(classifySensor('pch_cannonlake'), 'chipset')
})

test('groupSensors merges the user nvme set into CPU / board / NVMe', () => {
  const groups = groupSensors([
    { key: 'acpitz', temp: 28 },
    { key: 'nvme_composite', temp: 41 },
    { key: 'nvme_sensor_1', temp: 41 },
    { key: 'nvme_sensor_2', temp: 55 },
    { key: 'coretemp_package_id_0', temp: 45 },
  ])
  assert.deepEqual(groups.map((g) => g.kind), ['cpu', 'board', 'nvme'])
  assert.equal(groups[0].temp, 45)
  assert.equal(groups[1].temp, 28)
  assert.equal(groups[2].temp, 41)
  assert.equal(groups[2].hotspot, 55)
  assert.equal(groups[2].keys.includes('nvme_sensor_1'), false)
})

test('fallbackSensorName keeps unknown keys readable', () => {
  assert.equal(fallbackSensorName('foo_bar_baz'), 'foo bar baz')
})
