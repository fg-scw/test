import { Box, Paper, PaperProps, Switch, SwitchProps, Typography } from '@mui/material'
import { ReactNode } from 'react'
import FieldLabel from './FieldLabel'

export interface ToggleFieldProps extends Omit<SwitchProps, 'onChange'> {
  label: ReactNode
  description?: ReactNode
  tooltip?: ReactNode
  helperText?: ReactNode
  containerProps?: PaperProps
  onChange: SwitchProps['onChange']
}

export default function ToggleField({
  label,
  description,
  tooltip,
  helperText,
  containerProps,
  name,
  checked,
  onChange,
  ...switchProps
}: ToggleFieldProps) {
  return (
    <Paper
      variant="outlined"
      {...containerProps}
      sx={{
        width: '100%',
        p: 2,
        display: 'flex',
        flexDirection: 'column',
        gap: 1,
        ...containerProps?.sx
      }}
    >
      <Box display="flex" alignItems="center" justifyContent="space-between" gap={1.5}>
        <FieldLabel label={label} tooltip={tooltip} align="center" helperText={helperText} />
        <Switch name={name} checked={checked} onChange={onChange} {...switchProps} />
      </Box>
      {description ? (
        <Typography variant="caption" color="text.secondary">
          {description}
        </Typography>
      ) : null}
    </Paper>
  )
}
