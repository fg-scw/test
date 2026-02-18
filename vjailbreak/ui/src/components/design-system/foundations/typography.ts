import customTypography from 'src/theme/typography'

export const typography = customTypography

export type TypographyVariant = keyof typeof typography

export const fontFamilies = {
  sans: '"Fira Sans", "Inter", "Segoe UI", system-ui, -apple-system, BlinkMacSystemFont, "Helvetica Neue", sans-serif',
  mono: '"Fira Code", "SFMono-Regular", "Consolas", "Roboto Mono", monospace'
} as const

export const typographyMeta = {
  headings: ['h1', 'h2', 'h3', 'h4'] as const,
  subtitles: ['subtitle1', 'subtitle2'] as const,
  body: ['body1', 'body2', 'body3', 'body3SemiBold', 'body3Bold'] as const,
  captions: ['caption1', 'caption2', 'caption3', 'caption4'] as const
}
