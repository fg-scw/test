import { Box, TextField } from '@mui/material'
import { Controller, ControllerProps, FieldValues, useFormContext } from 'react-hook-form'
import { FieldLabel, FieldLabelProps } from 'src/components'

type ControllerRules = ControllerProps<FieldValues>['rules']

type FieldLabelCustomProps = Omit<FieldLabelProps, 'label' | 'helperText'>

export type RHFDateFieldProps = {
  name: ControllerProps<FieldValues>['name']
  rules?: ControllerRules
  label?: string
  disabled?: boolean
  helperText?: string
  error?: boolean
  labelProps?: FieldLabelCustomProps
  labelHelperText?: FieldLabelProps['helperText']
  placeholder?: string
}

export default function RHFDateField({
  name,
  rules,
  label,
  disabled,
  helperText,
  error: errorProp,
  labelProps,
  labelHelperText,
  placeholder,
  ...rest
}: RHFDateFieldProps) {
  const { control } = useFormContext()

  return (
    <Controller
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0 }}>
          {label ? (
            <FieldLabel
              label={label}
              helperText={labelHelperText}
              required={(rest as any).required}
              align="flex-start"
              {...labelProps}
            />
          ) : null}
          <TextField
            {...rest}
            {...field}
            type="date"
            label={undefined}
            placeholder={placeholder}
            disabled={disabled}
            error={fieldState.invalid || errorProp}
            helperText={fieldState.error?.message || helperText}
            size="small"
            variant="outlined"
            InputLabelProps={{
              shrink: true
            }}
          />
        </Box>
      )}
    />
  )
}
