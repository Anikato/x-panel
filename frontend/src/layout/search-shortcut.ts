export function searchShortcutLabel(platform = typeof navigator === 'undefined' ? '' : navigator.platform): string {
  return /mac/i.test(platform) ? '⌘K' : 'Ctrl+K'
}
