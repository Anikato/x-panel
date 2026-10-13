import assert from 'node:assert/strict'
import test from 'node:test'
import { getTermFontByKey } from './terminal-theme.ts'

test('terminal fonts keep the selected face and add the icon face before monospace', () => {
  const jetbrains = getTermFontByKey('jetbrains')
  assert.equal(
    jetbrains,
    "'JetBrains Mono', 'Fira Code', 'Cascadia Code', 'Consolas', 'MesloLGS Nerd Font Mono', monospace",
  )
  assert.equal(getTermFontByKey('system'), "'MesloLGS Nerd Font Mono', monospace")
  assert.equal(getTermFontByKey('missing'), jetbrains)
})
