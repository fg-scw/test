import { Box, Divider, Paper, PaperProps, Typography } from '@mui/material'
import { ReactNode } from 'react'

export interface SurfaceCardProps extends Omit<PaperProps, 'title' | 'variant'> {
  variant?: 'card' | 'section'
  title?: ReactNode
  subtitle?: ReactNode
  actions?: ReactNode
  footer?: ReactNode
  'data-testid'?: string
}

export default function SurfaceCard({
  variant = 'section',
  title,
  subtitle,
  actions,
  footer,
  children,
  sx,
  'data-testid': dataTestId = 'surface-card',
  ...paperProps
}: SurfaceCardProps) {
  const hasHeader = !!(title || subtitle || actions)
  const isSection = variant === 'section'

  return (
    <Paper
      elevation={0}
      {...paperProps}
      sx={{
        border: (theme) => (isSection ? 'none' : `1px solid ${theme.palette.divider}`),
        // boxShadow: (theme) => (isSection ? theme.shadows[6] : theme.shadows[3]),
        // borderBottom: (theme) =>
        //   isSection ? `1px solid ${theme.palette.primary.main}` : undefined,
        backgroundColor: (theme) => theme.palette.background.paper,
        display: 'flex',
        flexDirection: 'column',
        gap: isSection ? 1.5 : 2,
        p: isSection ? 2 : 3,
        //  paddingBottom: isSection ? 4 : 3,
        borderRadius: 0,
        ...sx
      }}
      data-testid={dataTestId}
    >
      {hasHeader ? (
        <Box data-testid={`${dataTestId}-header`} sx={{ display: 'flex', alignItems: 'center' }}>
          <Box sx={{ flex: 1 }}>
            {title ? (
              <Typography variant={isSection ? 'subtitle1' : 'h6'} component="h2">
                {title}
              </Typography>
            ) : null}
            {subtitle ? (
              <Typography variant="body2" color="text.secondary">
                {subtitle}
              </Typography>
            ) : null}
          </Box>
          {actions ? <Box sx={{ ml: 2 }}>{actions}</Box> : null}
        </Box>
      ) : null}

      {hasHeader && children && !isSection ? <Divider /> : null}

      {children ? (
        <Box
          data-testid={`${dataTestId}-body`}
          sx={{ display: 'flex', flexDirection: 'column', gap: isSection ? 1.5 : 2 }}
        >
          {children}
        </Box>
      ) : null}

      {footer ? (
        <Box data-testid={`${dataTestId}-footer`} sx={{ mt: hasHeader || children ? 1 : 0 }}>
          <Divider sx={{ mb: 2 }} />
          {footer}
        </Box>
      ) : null}
    </Paper>
  )
}
