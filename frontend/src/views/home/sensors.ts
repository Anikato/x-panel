import type { SensorTemp } from '@/api/interface'

export type SensorKind = 'cpu' | 'board' | 'nvme' | 'gpu' | 'wifi' | 'chipset' | 'other'

export interface SensorGroup {
  id: string
  kind: SensorKind
  temp: number
  hotspot?: number
  keys: string[]
}

const KIND_ORDER: SensorKind[] = ['cpu', 'board', 'nvme', 'gpu', 'chipset', 'wifi', 'other']

export function classifySensor(key: string): SensorKind {
  const k = key.toLowerCase()
  if (k.includes('nvme')) return 'nvme'
  if (k.includes('coretemp') || k.includes('k10temp') || k.includes('zenpower') || k.includes('cpu_thermal')) {
    return 'cpu'
  }
  if (k.includes('amdgpu') || k.includes('nouveau') || k.includes('radeon') || /(^|_)i915(_|$)/.test(k)) {
    return 'gpu'
  }
  if (k.includes('iwlwifi') || k.includes('wifi') || k.includes('ath10k') || k.includes('ath11k')) {
    return 'wifi'
  }
  if (k.includes('pch')) return 'chipset'
  if (k.includes('acpitz') || k.includes('thinkpad') || k.includes('dell_smm') || k.includes('iio_hwmon')) {
    return 'board'
  }
  return 'other'
}

export function sensorLabelKey(kind: SensorKind): string {
  const keys: Record<SensorKind, string> = {
    cpu: 'home.sensorCpu',
    board: 'home.sensorBoard',
    nvme: 'home.sensorNvme',
    gpu: 'home.sensorGpu',
    wifi: 'home.sensorWifi',
    chipset: 'home.sensorChipset',
    other: 'home.sensorTemp',
  }
  return keys[kind]
}

export function fallbackSensorName(key: string): string {
  return key.replace(/_/g, ' ')
}

export function groupSensors(sensors: SensorTemp[]): SensorGroup[] {
  const buckets = new Map<SensorKind, SensorTemp[]>()
  for (const sensor of sensors) {
    if (!sensor?.key || !Number.isFinite(sensor.temp)) continue
    const kind = classifySensor(sensor.key)
    const list = buckets.get(kind) ?? []
    list.push(sensor)
    buckets.set(kind, list)
  }

  const groups: SensorGroup[] = []
  for (const kind of KIND_ORDER) {
    const list = buckets.get(kind)
    if (!list?.length) continue
    if (kind === 'other') {
      for (const sensor of list) {
        groups.push({ id: sensor.key, kind, temp: sensor.temp, keys: [sensor.key] })
      }
      continue
    }
    if (kind === 'nvme') {
      groups.push(groupNvme(list))
      continue
    }
    const pack = list.find((s) => /package/i.test(s.key) || /tctl/i.test(s.key))
    const main = pack ?? hottest(list)
    groups.push({
      id: kind,
      kind,
      temp: main.temp,
      keys: list.map((s) => s.key),
    })
  }
  return groups
}

function hottest(list: SensorTemp[]): SensorTemp {
  return list.reduce((max, item) => (item.temp > max.temp ? item : max))
}

function groupNvme(list: SensorTemp[]): SensorGroup {
  const composite = list.find((s) => /composite/i.test(s.key))
  const main = composite ?? [...list].sort((a, b) => a.temp - b.temp)[0]
  const max = hottest(list)
  const hotspot = max.temp > main.temp + 1.5 ? max.temp : undefined
  const keys = list
    .filter((s) => s === main || (hotspot != null && s === max) || Math.abs(s.temp - main.temp) > 1)
    .map((s) => s.key)
  return {
    id: 'nvme',
    kind: 'nvme',
    temp: main.temp,
    hotspot,
    keys: keys.length ? keys : list.map((s) => s.key),
  }
}
