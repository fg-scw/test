import { Box, BoxProps, Typography } from '@mui/material'
import { ReactNode } from 'react'

import FieldLabel from './FieldLabel'

export type KeyValueItem = {
  label: string
  value?: ReactNode
}

export interface KeyValueGridProps extends Omit<BoxProps, 'children'> {
  items: KeyValueItem[]
  labelWidth?: number
}

export default function KeyValueGrid({ items, labelWidth = 220, ...boxProps }: KeyValueGridProps) {
  return (
    <Box
      sx={{
        display: 'grid',
        gridTemplateColumns: { xs: '1fr', sm: `${labelWidth}px 1fr` },
        columnGap: 2,
        rowGap: 1.5,
        alignItems: 'start'
      }}
      {...boxProps}
    >
      {items.map((item) => {
        const value = item.value === undefined || item.value === null || item.value === '' ? '—' : item.value

        return (
          <Box
            key={item.label}
            sx={{
              display: 'contents'
            }}
          >
            <FieldLabel label={item.label} />
            <Box
              sx={{
                display: 'flex',
                alignItems: 'flex-start',
                justifyContent: 'space-between',
                gap: 1,
                minWidth: 0
              }}
            >
              {typeof value === 'string' || typeof value === 'number' ? (
                <Typography variant="body2" sx={{ wordBreak: 'break-word', flex: 1 }}>
                  {value}
                </Typography>
              ) : (
                <Box sx={{ flex: 1, minWidth: 0 }}>{value}</Box>
              )}
            </Box>
          </Box>
        )
      })}
    </Box>
  )
}
