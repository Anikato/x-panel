export type IconSemantic =
  | 'nav.overview'
  | 'nav.files'
  | 'nav.terminal'
  | 'nav.website'
  | 'nav.network'
  | 'nav.container'
  | 'nav.certs'
  | 'nav.database'
  | 'nav.automation'
  | 'nav.monitor'
  | 'nav.security'
  | 'nav.logs'
  | 'nav.settings'
  | 'status.running'
  | 'status.stopped'
  | 'status.warning'
  | 'action.search'
  | 'action.add'
  | 'action.delete'
  | 'action.save'

const OUTLINE: Record<IconSemantic, string> = {
  'nav.overview': 'HomeFilled',
  'nav.files': 'FolderOpened',
  'nav.terminal': 'Monitor',
  'nav.website': 'ChromeFilled',
  'nav.network': 'Connection',
  'nav.container': 'Box',
  'nav.certs': 'Lock',
  'nav.database': 'Coin',
  'nav.automation': 'Timer',
  'nav.monitor': 'DataLine',
  'nav.security': 'Shield',
  'nav.logs': 'Document',
  'nav.settings': 'Setting',
  'status.running': 'CircleCheck',
  'status.stopped': 'CircleClose',
  'status.warning': 'Warning',
  'action.search': 'Search',
  'action.add': 'Plus',
  'action.delete': 'Delete',
  'action.save': 'Check',
}

const SOLID: Record<IconSemantic, string> = {
  ...OUTLINE,
  'nav.overview': 'House',
  'nav.files': 'Folder',
  'nav.terminal': 'Platform',
  'nav.website': 'Monitor',
  'nav.network': 'Link',
  'nav.container': 'Suitcase',
  'nav.certs': 'Lock',
  'nav.database': 'Coin',
  'nav.automation': 'AlarmClock',
  'nav.monitor': 'TrendCharts',
  'nav.settings': 'Tools',
  'status.running': 'SuccessFilled',
  'status.stopped': 'CircleCloseFilled',
  'status.warning': 'WarningFilled',
}

export function resolveSemanticIcon(name: IconSemantic, iconSet: 'outline' | 'solid' = 'outline'): string {
  const pack = iconSet === 'solid' ? SOLID : OUTLINE
  return pack[name] || OUTLINE[name] || 'QuestionFilled'
}

export function mapNavIcon(current: string, iconSet: 'outline' | 'solid'): string {
  if (current === 'Shield') return 'ShieldIcon'
  if (iconSet !== 'solid') return current
  const mapped: Record<string, string> = {
    HomeFilled: 'House',
    FolderOpened: 'Folder',
    Monitor: 'Platform',
    ChromeFilled: 'Monitor',
    Connection: 'Link',
    Box: 'Suitcase',
    DataLine: 'TrendCharts',
    Setting: 'Tools',
    Timer: 'AlarmClock',
    Shield: 'ShieldIcon',
  }
  return mapped[current] || current
}
