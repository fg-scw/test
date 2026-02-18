export const spacing = {
  xxs: 2,
  xs: 4,
  sm: 8,
  md: 12,
  lg: 16,
  xl: 24,
  xxl: 32,
  xxxl: 48
} as const

export type SpacingToken = keyof typeof spacing

export const spacingValue = (token: SpacingToken | number): number =>
  typeof token === 'number' ? token : spacing[token]

export const spacingPx = (token: SpacingToken | number): string => `${spacingValue(token)}px`

export const paddingStack = (
  vertical: SpacingToken | number,
  horizontal?: SpacingToken | number
): string => {
  const horizontalValue = horizontal ?? vertical
  return `${spacingPx(vertical)} ${spacingPx(horizontalValue)}`
}

export const gapStyle = (token: SpacingToken | number) => ({ gap: spacingPx(token) })
