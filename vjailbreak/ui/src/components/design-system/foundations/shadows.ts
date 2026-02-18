export const shadows = {
  none: 'none',
  hairline: '0 1px 1px rgba(15, 23, 42, 0.04)',
  xxs: '0 1px 2px rgba(15, 23, 42, 0.08)',
  xs: '0 2px 4px rgba(15, 23, 42, 0.1)',
  sm: '0 4px 8px rgba(15, 23, 42, 0.12)',
  md: '0 8px 16px rgba(15, 23, 42, 0.14)',
  lg: '0 12px 24px rgba(15, 23, 42, 0.16)',
  xl: '0 20px 32px rgba(15, 23, 42, 0.18)'
} as const

export type ShadowToken = keyof typeof shadows

export const shadowValue = (token: ShadowToken): string => shadows[token]
