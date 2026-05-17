const HISTORY_LIMIT = 200
const ARROW_UP = '\x1b[A'
const ARROW_DOWN = '\x1b[B'
const BACKSPACE = '\x7f'
const ENTER = '\r'
const CTRL_C = '\x03'
const CTRL_U = '\x15'
const CTRL_A = '\x01'
const CTRL_K = '\x0b'

const commandHistory: string[] = []

export interface TerminalHistoryController {
  handleData: (data: string, send: (data: string) => void, options?: { inAlternateBuffer?: boolean }) => boolean
  reset: () => void
}

const addCommand = (command: string) => {
  const normalized = command.trim()
  if (!normalized) return
  if (commandHistory[commandHistory.length - 1] === normalized) return
  commandHistory.push(normalized)
  if (commandHistory.length > HISTORY_LIMIT) commandHistory.shift()
}

export const createTerminalHistoryController = (): TerminalHistoryController => {
  let input = ''
  let cursor: number | null = null

  const replaceInput = (nextInput: string, send: (data: string) => void) => {
    input = nextInput
    send(`${CTRL_A}${CTRL_K}${nextInput}`)
  }

  const handleHistoryUp = (send: (data: string) => void) => {
    if (commandHistory.length === 0) return
    if (cursor === null) {
      cursor = commandHistory.length - 1
    } else if (cursor > 0) {
      cursor--
    }
    replaceInput(commandHistory[cursor], send)
  }

  const handleHistoryDown = (send: (data: string) => void) => {
    if (cursor === null) return
    if (cursor < commandHistory.length - 1) {
      cursor++
      replaceInput(commandHistory[cursor], send)
      return
    }
    cursor = null
    replaceInput('', send)
  }

  const trackInput = (data: string) => {
    cursor = null

    for (const char of data) {
      if (char === ENTER) {
        addCommand(input)
        input = ''
      } else if (char === BACKSPACE) {
        input = input.slice(0, -1)
      } else if (char === CTRL_C || char === CTRL_U) {
        input = ''
      } else if (char >= ' ') {
        input += char
      }
    }
  }

  return {
    handleData(data, send, options) {
      if (!options?.inAlternateBuffer && data === ARROW_UP) {
        handleHistoryUp(send)
        return true
      }
      if (!options?.inAlternateBuffer && data === ARROW_DOWN) {
        handleHistoryDown(send)
        return true
      }
      trackInput(data)
      return false
    },
    reset() {
      input = ''
      cursor = null
    },
  }
}
