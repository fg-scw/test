import { Box, Checkbox, FormControlLabel } from '@mui/material'
import { Controller, ControllerProps, FieldValues, useFormContext } from 'react-hook-form'

type ControllerRules = ControllerProps<FieldValues>['rules']

export type RHFCheckboxProps = {
  name: ControllerProps<FieldValues>['name']
  rules?: ControllerRules
  label?: string
  disabled?: boolean
  helperText?: string
  error?: boolean
  labelProps?: any
}

export default function RHFCheckbox({
  name,
  rules,
  label,
  disabled,
  helperText,
  error: errorProp,
  labelProps,
  ...rest
}: RHFCheckboxProps) {
  const { control } = useFormContext()

  return (
    <Controller
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0 }}>
          <FormControlLabel
            control={
              <Checkbox
                {...rest}
                {...field}
                checked={!!field.value}
                disabled={disabled}
                size="small"
              />
            }
            label={label}
            {...labelProps}
          />
          {fieldState.error?.message && (
            <Box sx={{ color: 'error.main', fontSize: '0.75rem', mt: 0.5, ml: 0.5 }}>
              {fieldState.error.message}
            </Box>
          )}
          {!fieldState.error && helperText && (
            <Box sx={{ color: 'text.secondary', fontSize: '0.75rem', mt: 0.5, ml: 0.5 }}>
              {helperText}
            </Box>
          )}
        </Box>
      )}
    />
  )
}
