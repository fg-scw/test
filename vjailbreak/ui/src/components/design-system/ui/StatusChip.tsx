import { Chip, ChipProps } from '@mui/material'
import { ReactNode } from 'react'

export type StatusChipTone = 'success' | 'error' | 'warning' | 'info' | 'default'

export interface StatusChipProps extends Omit<ChipProps, 'label' | 'color'> {
  label: ReactNode
  tone?: StatusChipTone
}

export default function StatusChip({ label, tone, ...props }: StatusChipProps) {
  const computedTone: StatusChipTone = (() => {
    if (tone !== undefined) return tone
    if (typeof label !== 'string') return 'default'

    const phaseLabel = label || 'Unknown'
    if (phaseLabel === 'Succeeded') return 'success'
    if (phaseLabel === 'Failed' || phaseLabel === 'ValidationFailed') return 'error'
    if (phaseLabel === 'Pending' || phaseLabel === 'Unknown') return 'default'
    return 'info'
  })()

  return <Chip label={label} color={computedTone as any} {...props} />
}
