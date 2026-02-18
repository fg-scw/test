import { Box, Typography } from '@mui/material'
import type { ReactNode } from 'react'

export interface TooltipContentProps {
  title?: ReactNode
  lines?: ReactNode[]
}

export default function TooltipContent({ title, lines }: TooltipContentProps) {
  return (
    <Box sx={{ display: 'grid', gap: 0.5 }}>
      {title ? (
        <Typography variant="subtitle2" component="div">
          {title}
        </Typography>
      ) : null}
      {lines?.length
        ? lines.map((line, idx) => (
            <Typography key={idx} variant="caption" component="div">
              {line}
            </Typography>
          ))
        : null}
    </Box>
  )
}
