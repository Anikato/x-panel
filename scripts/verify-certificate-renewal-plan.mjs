import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const root = resolve(import.meta.dirname, '..')
const checks = [
  ['frontend/src/api/interface/index.ts', ['CertificateRenewalPlanItem', 'renewalMetadataKnown', 'nextAutoRenewAt', 'nextSyncAt']],
  ['frontend/src/api/modules/ssl.ts', ['searchCertificateRenewalPlan', '/certificates/renewal-plan/search']],
  ['frontend/src/views/website/ssl/index.vue', ['name="renewal-plan"', 'loadRenewalPlan', 'renewalPlanRequestVersion', 'renewalPlanStatusType']],
  ['frontend/src/i18n/zh.ts', ["renewalPlan: '续签计划'", "nextAutoRenew: '预计续签'", "nextSync: '下次同步'"]],
]

for (const [path, tokens] of checks) {
  const content = readFileSync(resolve(root, path), 'utf8')
  for (const token of tokens) {
    if (!content.includes(token)) {
      throw new Error(`${path} is missing ${token}`)
    }
  }
}

console.log('certificate renewal plan frontend contract: OK')
