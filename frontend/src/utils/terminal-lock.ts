import { ElMessageBox } from 'element-plus'

type LockTab = {
  sessionId?: string
  hostId?: number
  locked?: boolean
  title: string
  closing?: boolean
  closedByUser?: boolean
}

type LockStore = {
  setLocked: (id: string, locked: boolean, name: string) => Promise<void>
  refresh: () => Promise<void>
}

// Unlock drops the session from the locked list before this function resumes.
// The tab watch closes a still-locked tab whose id has disappeared, so the
// flag has to be cleared first. Clearing it later also disconnects the
// browser, and an unlocked shell with no browser is killed.
export async function toggleBackgroundLock(
  tab: LockTab,
  store: LockStore,
  t: (key: string) => string,
) {
  if (!tab.sessionId || tab.hostId || tab.closing || tab.closedByUser) return
  const next = !tab.locked
  if (!next) {
    try {
      await ElMessageBox.confirm(t('terminal.unlockConfirm'), t('commons.tip'), {
        type: 'warning',
        confirmButtonText: t('terminal.unlockBackground'),
        cancelButtonText: t('commons.cancel'),
      })
    } catch {
      return
    }
    if (tab.closing || tab.closedByUser || !tab.sessionId) return
    tab.locked = false
  }
  try {
    await store.setLocked(tab.sessionId, next, tab.title)
  } catch {
    if (!next) tab.locked = true
    return
  }
  tab.locked = next
  await store.refresh().catch(() => {})
}
