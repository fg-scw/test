import { Box, BoxProps } from '@mui/material'
import { forwardRef } from 'react'

export interface RowProps extends BoxProps {
  children: React.ReactNode
  justifyContent?:
    | 'flex-start'
    | 'center'
    | 'flex-end'
    | 'space-between'
    | 'space-around'
    | 'space-evenly'
  alignItems?: 'flex-start' | 'center' | 'flex-end' | 'stretch'
  gap?: number
}

const Row = forwardRef<HTMLDivElement, RowProps>(function Row(
  { children, justifyContent = 'flex-start', alignItems = 'center', gap = 2, ...rest },
  ref
) {
  return (
    <Box
      ref={ref}
      sx={{
        display: 'flex',
        flexDirection: 'row',
        justifyContent,
        alignItems,
        gap
      }}
      {...rest}
    >
      {children}
    </Box>
  )
})

export default Row
