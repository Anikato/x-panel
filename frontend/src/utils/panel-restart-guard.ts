import { ElMessageBox } from 'element-plus'
import i18n from '@/i18n'
import { useGlobalStore } from '@/store/modules/global'
import { useTerminalSessionStore } from '@/store/modules/terminal-session'

// Returns true only when no background shell is still running.
// Choosing "去处理" opens the quick terminal and still returns false.
export async function confirmPanelProcessRestart(): Promise<boolean> {
  const sessions = useTerminalSessionStore()
  try {
    await sessions.refresh()
  } catch {
    return true
  }
  const running = sessions.items.filter((item) => item.status === 'running')
  if (running.length === 0) return true

  const t = i18n.global.t
  try {
    await ElMessageBox.confirm(
      String(t('terminal.backgroundBlock', { names: running.map((item) => item.name).join('、') })),
      String(t('terminal.backgroundBlockTitle')),
      {
        type: 'warning',
        confirmButtonText: String(t('terminal.backgroundHandle')),
        cancelButtonText: String(t('commons.cancel')),
      },
    )
  } catch {
    return false
  }
  const globalStore = useGlobalStore()
  globalStore.floatTermVisible = true
  globalStore.floatTermMinimized = false
  return false
}
