import { applyAccentPalette, generatePaletteFromHex, getPresetByKey } from '../utils/accent-colors.ts'
import { FONT_CDN } from './shared-tokens.ts'
import { emitAppearanceChanged } from './charts.ts'
import { cssImage, readEffectiveWallpaper, sanitizeWallpaperUrl } from './wallpaper-store.ts'
import type { ResolvedAppearance, UiFont } from './types.ts'

const loadedFonts = new Set<string>()

function loadCdnFont(url: string) {
  if (typeof document === 'undefined' || loadedFonts.has(url)) return
  loadedFonts.add(url)
  const link = document.createElement('link')
  link.rel = 'stylesheet'
  link.href = url
  document.head.appendChild(link)
}

export function applyResolvedAppearance(resolved: ResolvedAppearance): void {
  if (typeof document === 'undefined') return
  const root = document.documentElement
  const palette = resolved.accent.key === 'custom' && resolved.accent.custom
    ? generatePaletteFromHex(resolved.accent.custom)
    : getPresetByKey(resolved.accent.key)
  if (palette) applyAccentPalette(palette)

  for (const [prop, value] of Object.entries(resolved.cssVars)) {
    root.style.setProperty(prop, value)
  }

  root.classList.toggle('dark', resolved.colorMode === 'dark')
  root.classList.toggle('reduce-motion', resolved.reduceMotion)
  root.style.fontSize = resolved.densityTokens.fontSize

  const datasetKeys = [
    'theme', 'density', 'card-variant', 'sidebar-variant', 'subnav-variant',
    'icon-set', 'icon-container', 'transparency', 'color-mode', 'term-wallpaper',
    'chrome-texture', 'chrome-photo', 'term-follow',
  ]
  for (const key of datasetKeys) {
    const value = resolved.datasets[key]
    if (value) root.setAttribute(`data-${key}`, value)
    else root.removeAttribute(`data-${key}`)
  }

  const cdn = FONT_CDN[resolved.uiFontKey as UiFont]
  if (cdn) loadCdnFont(cdn)

  const chromePhoto = resolved.chromeImageMode === 'upload'
    ? readEffectiveWallpaper('chrome', resolved.themeId)
    : resolved.chromeImageMode === 'url' ? sanitizeWallpaperUrl(resolved.chromeImageUrl) : ''
  const termPhoto = resolved.termFollowChrome
    ? chromePhoto
    : resolved.termImageMode === 'upload'
      ? readEffectiveWallpaper('term', resolved.themeId)
      : resolved.termImageMode === 'url' ? sanitizeWallpaperUrl(resolved.termImageUrl) : ''
  root.style.setProperty('--xp-chrome-photo', cssImage(chromePhoto))
  root.style.setProperty('--xp-term-photo', cssImage(termPhoto))
  emitAppearanceChanged()
}

export function readResolveEnv(): { systemDark: boolean; systemReduceMotion: boolean; transparencySupported: boolean } {
  if (typeof window === 'undefined') {
    return { systemDark: true, systemReduceMotion: false, transparencySupported: true }
  }
  const transparencySupported = typeof CSS !== 'undefined' && typeof CSS.supports === 'function'
    ? CSS.supports('filter', 'blur(8px)')
    : true
  return {
    systemDark: window.matchMedia('(prefers-color-scheme: dark)').matches,
    systemReduceMotion: window.matchMedia('(prefers-reduced-motion: reduce)').matches,
    transparencySupported,
  }
}
