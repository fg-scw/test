import { Box, RadioGroup, FormControlLabel, Radio, FormLabel } from '@mui/material'
import { Controller, ControllerProps, FieldValues, useFormContext } from 'react-hook-form'

type ControllerRules = ControllerProps<FieldValues>['rules']

export interface Option {
  label: string
  value: any
}

export type RHFRadioGroupProps = {
  name: ControllerProps<FieldValues>['name']
  options: Option[]
  rules?: ControllerRules
  label?: string
  disabled?: boolean
  helperText?: string
  error?: boolean
  labelProps?: any
  row?: boolean
}

export default function RHFRadioGroup({
  name,
  options,
  rules,
  label,
  disabled,
  helperText,
  error: errorProp,
  labelProps,
  row = false,
  ...rest
}: RHFRadioGroupProps) {
  const { control } = useFormContext()

  return (
    <Controller
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0 }}>
          {label && <FormLabel component="legend">{label}</FormLabel>}
          <RadioGroup {...rest} {...field} value={field.value ?? ''} row={row}>
            {options.map((option) => (
              <FormControlLabel
                key={String(option.value)}
                value={option.value}
                control={<Radio size="small" disabled={disabled} />}
                label={option.label}
              />
            ))}
          </RadioGroup>
          {fieldState.error?.message && (
            <Box sx={{ color: 'error.main', fontSize: '0.75rem', mt: 0.5 }}>
              {fieldState.error.message}
            </Box>
          )}
          {!fieldState.error && helperText && (
            <Box sx={{ color: 'text.secondary', fontSize: '0.75rem', mt: 0.5 }}>{helperText}</Box>
          )}
        </Box>
      )}
    />
  )
}
