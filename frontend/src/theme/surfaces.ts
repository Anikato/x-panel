import { BLACK, WHITE, mixSrgb } from './pack-color.ts'
import type { SemanticColors, SurfacePreset } from './types.ts'

const LIGHT_C1 = { bg: '#E4E6EC', surface: '#F4F5F8' }
const LIGHT_TINTS: Record<SurfacePreset, string> = {
  graphite: '#64748B',
  abyss: '#1D4ED8',
  void: '#000000',
  tinted: '#0F766E',
  cosmos: '#7E22CE',
  warm: '#92400E',
}

export function lightSurfaceFamily(key: SurfacePreset): SurfacePresetDef['colors'] {
  const tint = LIGHT_TINTS[key]
  const bg = mixSrgb(LIGHT_C1.bg, tint, 0.06)
  const surface = mixSrgb(LIGHT_C1.surface, tint, 0.03)
  const elevated = mixSrgb(surface, WHITE, 0.4)
  const inset = mixSrgb(surface, BLACK, 0.04)
  const sidebar = mixSrgb(bg, BLACK, 0.04)
  return {
    bgBase: bg,
    bgSurface: surface,
    bgElevated: elevated,
    bgOverlay: elevated,
    bgSidebar: sidebar,
    bgHeader: sidebar,
    bgInput: inset,
    bgTableHeader: inset,
    bgInset: inset,
    bgAuth: mixSrgb(bg, BLACK, 0.04),
  }
}

export interface SurfacePresetDef {
  key: SurfacePreset
  name: string
  preview: string
  colors: Pick<SemanticColors, 'bgBase' | 'bgSurface' | 'bgElevated' | 'bgOverlay' | 'bgSidebar' | 'bgHeader' | 'bgInput' | 'bgTableHeader' | 'bgInset' | 'bgAuth'>
}

export const SURFACE_PRESET_DEFS: SurfacePresetDef[] = [
  {
    key: 'graphite',
    name: '石墨',
    preview: 'linear-gradient(160deg, #111318, #191C23)',
    colors: {
      bgBase: '#111318',
      bgSurface: '#191C23',
      bgElevated: '#222630',
      bgOverlay: '#222630',
      bgSidebar: '#0E1016',
      bgHeader: '#111318',
      bgInput: '#151821',
      bgTableHeader: '#151821',
      bgInset: '#151821',
      bgAuth: '#0B0D12',
    },
  },
  {
    key: 'abyss',
    name: '深渊',
    preview: 'linear-gradient(160deg, #080a10, #0e1420)',
    colors: {
      bgBase: '#080a10',
      bgSurface: '#141c2b',
      bgElevated: '#1c2638',
      bgOverlay: '#1e293b',
      bgSidebar: '#060810',
      bgHeader: '#080a10',
      bgInput: '#0f172a',
      bgTableHeader: '#0f172a',
      bgInset: '#0f172a',
      bgAuth: '#060810',
    },
  },
  {
    key: 'void',
    name: '纯黑',
    preview: 'linear-gradient(160deg, #000000, #0a0a0a)',
    colors: {
      bgBase: '#050508',
      bgSurface: '#111115',
      bgElevated: '#1a1a1e',
      bgOverlay: '#222226',
      bgSidebar: '#020204',
      bgHeader: '#050508',
      bgInput: '#0a0a0e',
      bgTableHeader: '#0a0a0e',
      bgInset: '#0a0a0e',
      bgAuth: '#020204',
    },
  },
  {
    key: 'tinted',
    name: '微染',
    preview: 'linear-gradient(160deg, #080a10, #0a1510)',
    colors: {
      bgBase: '#080a10',
      bgSurface: '#121c28',
      bgElevated: '#1a2634',
      bgOverlay: '#1e2e3b',
      bgSidebar: '#060810',
      bgHeader: '#080a10',
      bgInput: '#0d1620',
      bgTableHeader: '#0d1620',
      bgInset: '#0d1620',
      bgAuth: '#060810',
    },
  },
  {
    key: 'cosmos',
    name: '星空',
    preview: 'linear-gradient(160deg, #0a0818, #14102a)',
    colors: {
      bgBase: '#0a0818',
      bgSurface: '#161230',
      bgElevated: '#1e1840',
      bgOverlay: '#252048',
      bgSidebar: '#06050f',
      bgHeader: '#0a0818',
      bgInput: '#100e24',
      bgTableHeader: '#100e24',
      bgInset: '#100e24',
      bgAuth: '#06050f',
    },
  },
  {
    key: 'warm',
    name: '暖夜',
    preview: 'linear-gradient(160deg, #100c08, #1a1410)',
    colors: {
      bgBase: '#100c08',
      bgSurface: '#1c1610',
      bgElevated: '#262018',
      bgOverlay: '#302820',
      bgSidebar: '#0a0806',
      bgHeader: '#100c08',
      bgInput: '#14100c',
      bgTableHeader: '#14100c',
      bgInset: '#14100c',
      bgAuth: '#0a0806',
    },
  },
]

export function getSurfacePreset(key: string): SurfacePresetDef | undefined {
  return SURFACE_PRESET_DEFS.find((item) => item.key === key)
}
