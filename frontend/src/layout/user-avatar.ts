export const AVATAR_COUNT = 8

export function avatarIndex(name: string): number {
  const s = name || 'user'
  let hash = 2166136261
  for (let i = 0; i < s.length; i++) {
    hash ^= s.charCodeAt(i)
    hash = Math.imul(hash, 16777619)
  }
  return (hash >>> 0) % AVATAR_COUNT
}
