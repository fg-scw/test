import { Box, BoxProps } from '@mui/material'
import { forwardRef } from 'react'

export interface SectionProps extends BoxProps {
  children: React.ReactNode
}

const Section = forwardRef<HTMLDivElement, SectionProps>(function Section(
  { children, ...rest },
  ref
) {
  return (
    <Box ref={ref} sx={{ display: 'flex', flexDirection: 'column', gap: 2 }} {...rest}>
      {children}
    </Box>
  )
})

export default Section
