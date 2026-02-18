import { Box, Typography, BoxProps } from '@mui/material'
import { forwardRef } from 'react'
import type { ReactNode } from 'react'

export interface SectionHeaderProps extends BoxProps {
  title?: string
  subtitle?: string
  actions?: ReactNode
}

const SectionHeader = forwardRef<HTMLDivElement, SectionHeaderProps>(function SectionHeader(
  { title, subtitle, actions, ...rest },
  ref
) {
  return (
    <Box ref={ref} sx={{ mb: 2, display: 'flex', alignItems: 'flex-start', gap: 2 }} {...rest}>
      <Box sx={{ flex: 1 }}>
        {title && (
          <Typography variant="h6" component="h2" sx={{ mb: subtitle ? 1 : 0 }}>
            {title}
          </Typography>
        )}
        {subtitle && (
          <Typography variant="body2" color="text.secondary">
            {subtitle}
          </Typography>
        )}
      </Box>
      {actions ? <Box sx={{ mt: title ? 0.25 : 0 }}>{actions}</Box> : null}
    </Box>
  )
})

export default SectionHeader
