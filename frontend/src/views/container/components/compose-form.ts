const nameRe = /^[a-zA-Z0-9][a-zA-Z0-9_.-]*$/

export function isValidComposeName(name: string): boolean {
  return nameRe.test(name)
}

export function composeCreateMode(input: { content: string; path: string }): 'create' | 'attach' | 'invalid' {
  const content = input.content.trim()
  const path = input.path.trim()
  if (content && !path) return 'create'
  if (!content && path) return 'attach'
  return 'invalid'
}
