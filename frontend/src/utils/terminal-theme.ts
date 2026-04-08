import type { ITheme } from '@xterm/xterm'

export interface TermThemePreset {
  key: string
  name: string
  theme: ITheme
}

const defaultTheme: ITheme = {
  background: '#0b0e14',
  foreground: '#e6edf3',
  cursor: '#22d3ee',
  cursorAccent: '#0b0e14',
  selectionBackground: 'rgba(34, 211, 238, 0.2)',
  black: '#0b0e14',
  red: '#f87171',
  green: '#4ade80',
  yellow: '#fbbf24',
  blue: '#60a5fa',
  magenta: '#c084fc',
  cyan: '#22d3ee',
  white: '#e6edf3',
  brightBlack: '#475569',
  brightRed: '#fca5a5',
  brightGreen: '#86efac',
  brightYellow: '#fde68a',
  brightBlue: '#93c5fd',
  brightMagenta: '#d8b4fe',
  brightCyan: '#67e8f9',
  brightWhite: '#f8fafc',
}

const draculaTheme: ITheme = {
  background: '#282a36',
  foreground: '#f8f8f2',
  cursor: '#f8f8f2',
  cursorAccent: '#282a36',
  selectionBackground: 'rgba(68, 71, 90, 0.5)',
  black: '#21222c',
  red: '#ff5555',
  green: '#50fa7b',
  yellow: '#f1fa8c',
  blue: '#bd93f9',
  magenta: '#ff79c6',
  cyan: '#8be9fd',
  white: '#f8f8f2',
  brightBlack: '#6272a4',
  brightRed: '#ff6e6e',
  brightGreen: '#69ff94',
  brightYellow: '#ffffa5',
  brightBlue: '#d6acff',
  brightMagenta: '#ff92df',
  brightCyan: '#a4ffff',
  brightWhite: '#ffffff',
}

const oneDarkTheme: ITheme = {
  background: '#282c34',
  foreground: '#abb2bf',
  cursor: '#528bff',
  cursorAccent: '#282c34',
  selectionBackground: 'rgba(62, 68, 81, 0.5)',
  black: '#282c34',
  red: '#e06c75',
  green: '#98c379',
  yellow: '#e5c07b',
  blue: '#61afef',
  magenta: '#c678dd',
  cyan: '#56b6c2',
  white: '#abb2bf',
  brightBlack: '#5c6370',
  brightRed: '#e06c75',
  brightGreen: '#98c379',
  brightYellow: '#e5c07b',
  brightBlue: '#61afef',
  brightMagenta: '#c678dd',
  brightCyan: '#56b6c2',
  brightWhite: '#ffffff',
}

const solarizedDarkTheme: ITheme = {
  background: '#002b36',
  foreground: '#839496',
  cursor: '#93a1a1',
  cursorAccent: '#002b36',
  selectionBackground: 'rgba(7, 54, 66, 0.6)',
  black: '#073642',
  red: '#dc322f',
  green: '#859900',
  yellow: '#b58900',
  blue: '#268bd2',
  magenta: '#d33682',
  cyan: '#2aa198',
  white: '#eee8d5',
  brightBlack: '#586e75',
  brightRed: '#cb4b16',
  brightGreen: '#586e75',
  brightYellow: '#657b83',
  brightBlue: '#839496',
  brightMagenta: '#6c71c4',
  brightCyan: '#93a1a1',
  brightWhite: '#fdf6e3',
}

const monokaiTheme: ITheme = {
  background: '#272822',
  foreground: '#f8f8f2',
  cursor: '#f8f8f0',
  cursorAccent: '#272822',
  selectionBackground: 'rgba(73, 72, 62, 0.5)',
  black: '#272822',
  red: '#f92672',
  green: '#a6e22e',
  yellow: '#f4bf75',
  blue: '#66d9ef',
  magenta: '#ae81ff',
  cyan: '#a1efe4',
  white: '#f8f8f2',
  brightBlack: '#75715e',
  brightRed: '#f92672',
  brightGreen: '#a6e22e',
  brightYellow: '#f4bf75',
  brightBlue: '#66d9ef',
  brightMagenta: '#ae81ff',
  brightCyan: '#a1efe4',
  brightWhite: '#f9f8f5',
}

export const TERMINAL_THEME_PRESETS: TermThemePreset[] = [
  { key: 'default', name: 'X-Panel', theme: defaultTheme },
  { key: 'dracula', name: 'Dracula', theme: draculaTheme },
  { key: 'onedark', name: 'One Dark', theme: oneDarkTheme },
  { key: 'solarized', name: 'Solarized Dark', theme: solarizedDarkTheme },
  { key: 'monokai', name: 'Monokai', theme: monokaiTheme },
]

export const TERMINAL_FONT_PRESETS = [
  { key: 'jetbrains', name: 'JetBrains Mono', family: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', 'Consolas', monospace" },
  { key: 'firacode', name: 'Fira Code', family: "'Fira Code', 'JetBrains Mono', 'Consolas', monospace" },
  { key: 'cascadia', name: 'Cascadia Code', family: "'Cascadia Code', 'JetBrains Mono', 'Consolas', monospace" },
  { key: 'consolas', name: 'Consolas', family: "'Consolas', 'Courier New', monospace" },
  { key: 'system', name: '系统等宽', family: "monospace" },
]

export function getTermThemeByKey(key: string): ITheme {
  return TERMINAL_THEME_PRESETS.find(p => p.key === key)?.theme || defaultTheme
}

export function getTermFontByKey(key: string): string {
  return TERMINAL_FONT_PRESETS.find(p => p.key === key)?.family || TERMINAL_FONT_PRESETS[0].family
}

export function applyBgOpacity(theme: ITheme, opacity: number): ITheme {
  if (opacity >= 1) return theme
  const bg = theme.background || '#000000'
  const hex = bg.replace('#', '')
  const r = parseInt(hex.slice(0, 2), 16)
  const g = parseInt(hex.slice(2, 4), 16)
  const b = parseInt(hex.slice(4, 6), 16)
  return { ...theme, background: `rgba(${r}, ${g}, ${b}, ${opacity})` }
}

export const terminalTheme = defaultTheme
