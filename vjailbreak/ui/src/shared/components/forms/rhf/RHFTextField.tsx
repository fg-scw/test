import { Box } from '@mui/material'
import { Controller, ControllerProps, FieldValues, useFormContext } from 'react-hook-form'
import { FieldLabel, FieldLabelProps } from 'src/components'
import TextField, { TextFieldProps } from '../TextField'

type ControllerRules = ControllerProps<FieldValues>['rules']

type FieldLabelCustomProps = Omit<FieldLabelProps, 'label' | 'helperText'>

export type RHFTextFieldProps = TextFieldProps & {
  name: ControllerProps<FieldValues>['name']
  rules?: ControllerRules
  labelHelperText?: FieldLabelProps['helperText']
  labelProps?: FieldLabelCustomProps
  disabled?: boolean
  onValueChange?: (
    value: string,
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>
  ) => void
}

export default function RHFTextField({
  name,
  rules,
  helperText,
  error: errorProp,
  label,
  labelHelperText,
  labelProps,
  onValueChange,
  disabled = false,
  ...rest
}: RHFTextFieldProps) {
  const { control } = useFormContext()

  return (
    <Controller
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
          {label ? (
            <FieldLabel
              label={label}
              helperText={labelHelperText}
              required={rest.required}
              align="flex-start"
              {...labelProps}
            />
          ) : null}
          <TextField
            {...rest}
            {...field}
            label={undefined}
            value={field.value ?? ''}
            onChange={(event) => {
              field.onChange(event)
              onValueChange?.(event.target.value, event)
            }}
            error={fieldState.invalid || errorProp}
            helperText={fieldState.error?.message ?? helperText}
            disabled={disabled}
          />
        </Box>
      )}
    />
  )
}
