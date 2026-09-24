const domainSeparator = /[,，;；\s]+/u

export function parseDomainList(value: string): string[] {
  const seen = new Set<string>()
  const domains: string[] = []
  for (const part of (value || '').split(domainSeparator)) {
    const domain = part.trim().replace(/^\.+|\.+$/g, '')
    if (!domain || seen.has(domain)) continue
    seen.add(domain)
    domains.push(domain)
  }
  return domains
}

export function joinDomainList(domains: string[]): string {
  return parseDomainList(domains.join(',')).join(',')
}
