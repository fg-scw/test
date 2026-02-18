import { useMemo } from 'react'
import { Autocomplete, Box, Checkbox, Chip } from '@mui/material'
import { Controller, ControllerProps, FieldValues, useFormContext } from 'react-hook-form'
import { FieldLabel, FieldLabelProps } from 'src/components'
import TextField from '../TextField'

type ControllerRules = ControllerProps<FieldValues>['rules']

type FieldLabelCustomProps = Omit<FieldLabelProps, 'label' | 'helperText'>

export type RHFAutocompleteProps<TOption> = {
  name: ControllerProps<FieldValues>['name']
  options: TOption[]
  multiple?: boolean
  label?: string
  labelHelperText?: FieldLabelProps['helperText']
  labelProps?: FieldLabelCustomProps
  placeholder?: string
  disabled?: boolean
  helperText?: string
  error?: boolean
  rules?: ControllerRules
  getOptionLabel: (option: TOption) => string
  getOptionValue: (option: TOption) => string
  renderOptionLabel?: (option: TOption) => string
  showCheckboxes?: boolean
  onValueChange?: (value: string[] | string) => void
  'data-testid'?: string
}

export default function RHFAutocomplete<TOption>({
  name,
  options,
  multiple = false,
  label,
  labelHelperText,
  labelProps,
  placeholder,
  disabled,
  helperText,
  error: errorProp,
  rules,
  getOptionLabel,
  getOptionValue,
  renderOptionLabel,
  showCheckboxes = false,
  onValueChange,
  'data-testid': dataTestId = 'rhf-autocomplete'
}: RHFAutocompleteProps<TOption>) {
  const { control } = useFormContext()

  const optionById = useMemo(() => {
    const map = new Map<string, TOption>()
    options.forEach((o) => map.set(getOptionValue(o), o))
    return map
  }, [options, getOptionValue])

  return (
    <Controller
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => {
        const rawValue = field.value

        const selectedOptions = multiple
          ? (Array.isArray(rawValue) ? rawValue : [])
              .map((id) => optionById.get(String(id)))
              .filter(Boolean)
          : rawValue
            ? (optionById.get(String(rawValue)) ?? null)
            : null

        return (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }} data-testid={dataTestId}>
            {label ? (
              <FieldLabel
                label={label}
                helperText={labelHelperText}
                required={false}
                align="flex-start"
                {...labelProps}
              />
            ) : null}

            <Autocomplete
              multiple={multiple}
              options={options}
              disabled={disabled}
              value={selectedOptions as any}
              onChange={(_e, value) => {
                if (multiple) {
                  const ids = (value as TOption[]).map((v) => getOptionValue(v))
                  field.onChange(ids)
                  onValueChange?.(ids)
                  return
                }

                const id = value ? getOptionValue(value as TOption) : ''
                field.onChange(id)
                onValueChange?.(id)
              }}
              isOptionEqualToValue={(option, value) =>
                getOptionValue(option) === getOptionValue(value as TOption)
              }
              getOptionLabel={(option) => getOptionLabel(option)}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label={undefined}
                  placeholder={placeholder}
                  error={fieldState.invalid || errorProp}
                  helperText={fieldState.error?.message ?? helperText}
                />
              )}
              renderTags={(value, getTagProps) =>
                (value as TOption[]).map((option, index) => (
                  <Chip
                    {...getTagProps({ index })}
                    key={getOptionValue(option)}
                    label={renderOptionLabel ? renderOptionLabel(option) : getOptionLabel(option)}
                    size="small"
                  />
                ))
              }
              renderOption={(props, option, state) => (
                <li {...props}>
                  {showCheckboxes && multiple ? (
                    <Checkbox style={{ marginRight: 8 }} checked={state.selected} size="small" />
                  ) : null}
                  {renderOptionLabel ? renderOptionLabel(option) : getOptionLabel(option)}
                </li>
              )}
              disableCloseOnSelect={multiple}
              size="small"
              sx={{ width: '100%' }}
            />
          </Box>
        )
      }}
    />
  )
}
