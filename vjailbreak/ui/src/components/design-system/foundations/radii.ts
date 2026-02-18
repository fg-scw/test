export const radii = {
  none: 0,
  hairline: 1,
  xs: 2,
  sm: 4,
  md: 8,
  lg: 12,
  xl: 16,
  pill: 999,
  round: '50%'
} as const

export type RadiusToken = keyof typeof radii

export const radiusValue = (token: RadiusToken | number | string): number | string =>
  typeof token === 'number' || typeof token === 'string' ? token : radii[token]
