export interface TerminalControl {
  kind: 'control'
  op: string
  id: string
  message: string
}

export type TerminalServerFrame =
  | { kind: 'output'; bytes: Uint8Array }
  | { kind: 'output'; text: string }
  | TerminalControl

export function encodeTerminalDetach(name: string): Uint8Array {
  const body = new TextEncoder().encode(JSON.stringify({ op: 'detach', name }))
  const frame = new Uint8Array(body.length + 1)
  frame[0] = 2
  frame.set(body, 1)
  return frame
}

export function readTerminalServerFrame(data: string | ArrayBuffer): TerminalServerFrame {
  if (typeof data !== 'string') {
    return { kind: 'output', bytes: new Uint8Array(data) }
  }
  const trimmed = data.trim()
  if (trimmed.startsWith('{')) {
    try {
      const parsed = JSON.parse(trimmed) as { op?: unknown; id?: unknown; message?: unknown }
      if (parsed && typeof parsed.op === 'string') {
        return {
          kind: 'control',
          op: parsed.op,
          id: typeof parsed.id === 'string' ? parsed.id : '',
          message: typeof parsed.message === 'string' ? parsed.message : '',
        }
      }
    } catch {
      // Shell output can look almost like JSON. Keep it on screen.
    }
  }
  return { kind: 'output', text: data }
}
