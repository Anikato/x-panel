import assert from 'node:assert/strict'
import test from 'node:test'

import zh from '../../i18n/zh.ts'

test('cloud control visible labels do not expose the upstream brand', () => {
  assert.equal(zh.menu.nezhaAgent, '云控面板')
  assert.equal(zh.nezhaAgent.title, '云控面板')
  assert.doesNotMatch(zh.nezhaAgent.pageDesc, /哪吒|Nezha/i)
  assert.doesNotMatch(zh.nezhaAgent.logsTitle, /哪吒|Nezha/i)
  assert.doesNotMatch(zh.nezhaAgent.alert.componentMissing, /哪吒|Nezha/i)
  assert.doesNotMatch(zh.nezhaAgent.alert.conflicts, /哪吒|Nezha/i)
})
