export type ReachStatus = 'online' | 'warning' | 'offline'

export function reachStatus(failStreak: number): ReachStatus {
  if (failStreak <= 0) return 'online'
  if (failStreak === 1) return 'warning'
  return 'offline'
}
