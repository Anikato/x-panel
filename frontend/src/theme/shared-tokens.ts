import type { Density, HeaderHeight, RadiusPreset, SidebarWidth, UiFont } from './types.ts'
import type { DensityTokens, MotionTokens, ShapeTokens } from './types.ts'

export const CUSTOM_UI_FONT_FAMILY = 'XP Custom UI'

export const UI_FONT_STACKS: Record<UiFont, string> = {
  system: "ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Noto Sans CJK SC', 'Noto Sans SC', 'Noto Sans', 'DejaVu Sans', 'Liberation Sans', 'PingFang SC', 'Microsoft YaHei', sans-serif",
  inter: "'Inter', ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Noto Sans SC', 'PingFang SC', 'Microsoft YaHei', sans-serif",
  noto: "'Noto Sans SC', ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, 'PingFang SC', 'Microsoft YaHei', sans-serif",
  lxgw: "'LXGW WenKai', ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, 'PingFang SC', 'Microsoft YaHei', sans-serif",
  custom: `'${CUSTOM_UI_FONT_FAMILY}', ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Noto Sans SC', 'PingFang SC', 'Microsoft YaHei', sans-serif`,
}

export const MONO_FONT_STACK = "'JetBrains Mono', 'Fira Code', 'Cascadia Code', 'Consolas', ui-monospace, monospace"

export const DENSITY_TOKENS: Record<Density, DensityTokens> = {
  compact: { fontSize: '13px', spacing: '16px', formMargin: '16px', controlHeight: '32px', rowHeight: '36px' },
  default: { fontSize: '14px', spacing: '24px', formMargin: '20px', controlHeight: '36px', rowHeight: '44px' },
  comfortable: { fontSize: '15px', spacing: '24px', formMargin: '24px', controlHeight: '40px', rowHeight: '52px' },
}

export const RADIUS_TOKENS: Record<RadiusPreset, ShapeTokens> = {
  sharp: { radiusControl: '2px', radiusCard: '4px', radiusOverlay: '6px' },
  default: { radiusControl: '6px', radiusCard: '10px', radiusOverlay: '12px' },
  rounded: { radiusControl: '10px', radiusCard: '16px', radiusOverlay: '18px' },
}

export const SIDEBAR_WIDTH_TOKENS: Record<SidebarWidth, string> = {
  narrow: '200px',
  default: '224px',
  wide: '256px',
}

export const HEADER_HEIGHT_TOKENS: Record<HeaderHeight, string> = {
  compact: '48px',
  default: '56px',
  comfortable: '64px',
}

export const DEFAULT_MOTION: MotionTokens = {
  hoverMs: 100,
  pageMs: 80,
  menuMs: 140,
  drawerMs: 180,
  ease: 'cubic-bezier(0.2, 0, 0, 1)',
}

export const FONT_CDN: Partial<Record<UiFont, string>> = {
  lxgw: 'https://cdn.jsdelivr.net/npm/lxgw-wenkai-webfont@1.7.0/style.css',
  inter: 'https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap',
  noto: 'https://fonts.googleapis.com/css2?family=Noto+Sans+SC:wght@400;500;700&display=swap',
}
