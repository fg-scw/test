import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined'
import { Box, Tooltip, TooltipProps, Typography } from '@mui/material'
import { ReactNode } from 'react'

export interface FieldLabelProps {
  label: ReactNode
  tooltip?: ReactNode
  required?: boolean
  align?: 'flex-start' | 'center'
  helperText?: ReactNode
  tooltipPlacement?: TooltipProps['placement']
}

const tooltipSlotProps: TooltipProps['componentsProps'] = {
  tooltip: {
    sx: {
      fontSize: 12,
      lineHeight: 1.5,
      letterSpacing: 0,
      maxWidth: 260,
      px: 1.5,
      py: 1
    }
  }
}

export function FieldLabel({
  label,
  tooltip,
  required,
  align = 'center',
  helperText,
  tooltipPlacement = 'top'
}: FieldLabelProps) {
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.25 }}>
      <Box
        sx={{
          display: 'flex',
          alignItems: align,
          gap: 0.75,
          color: 'text.primary'
        }}
      >
        <Typography variant="body2" fontWeight={600} component="span">
          {label}
          {required ? (
            <Box component="span" sx={{ color: 'error.main', ml: 0.5 }}>
              *
            </Box>
          ) : null}
        </Typography>
        {tooltip ? (
          <Tooltip
            placement={tooltipPlacement}
            arrow
            componentsProps={tooltipSlotProps}
            title={
              typeof tooltip === 'string' ? (
                <Typography variant="caption" sx={{ fontSize: 12, lineHeight: 1.4 }}>
                  {tooltip}
                </Typography>
              ) : (
                tooltip
              )
            }
          >
            <InfoOutlinedIcon sx={{ fontSize: 18, color: 'text.secondary', cursor: 'help' }} />
          </Tooltip>
        ) : null}
      </Box>
      {helperText ? (
        <Typography variant="caption" color="text.secondary">
          {helperText}
        </Typography>
      ) : null}
    </Box>
  )
}

export default FieldLabel
