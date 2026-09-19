import type { NetInterface, NetIOInfo } from '@/api/interface'

const NOISE_IFACE = /^(lo|docker\d*|docker_gwbridge|br-[0-9a-f]{6,}|veth|fwbr|fwln|fwpr|tap|tunl|cni|flannel|cali|virbr|vnet|dummy)/i

export function isNoiseIface(name: string): boolean {
  return NOISE_IFACE.test(name)
}

export function isLinkLocalDNS(server: string): boolean {
  const host = server.split('%')[0].toLowerCase()
  return host.startsWith('fe80:') || host === '::1' || host.startsWith('127.')
}

export function filterDNS(servers: string[] | undefined): string[] {
  return (servers || []).filter((s) => s && !isLinkLocalDNS(s))
}

export function filterAddressIfaces(ifaces: NetInterface[] | undefined): NetInterface[] {
  return (ifaces || []).filter((iface) => {
    if (!iface?.name || isNoiseIface(iface.name)) return false
    return Boolean(iface.ipv4?.length)
  })
}

export function filterInventoryNics(ifaces: NetInterface[] | undefined): NetInterface[] {
  return (ifaces || []).filter((iface) => iface?.name && !isNoiseIface(iface.name) && !/^(tun\d*|tailscale)/i.test(iface.name))
}

export function formatLinkSpeed(mbps?: number): string {
  if (!mbps || mbps <= 0) return ''
  if (mbps >= 1000) {
    const gbps = mbps / 1000
    const label = Number.isInteger(gbps) ? String(gbps) : String(gbps)
    return `${label} Gbps`
  }
  return `${mbps} Mbps`
}

export function filterTrafficNics(nics: NetIOInfo[] | undefined): NetIOInfo[] {
  const cleaned = (nics || []).filter((nic) => nic?.name && nic.name !== 'lo' && !isNoiseIface(nic.name))
  const active = cleaned.filter((nic) => (nic.speedUp || 0) + (nic.speedDown || 0) >= 64)
  if (active.length) return active.slice(0, 6)
  return cleaned.slice(0, 3)
}
