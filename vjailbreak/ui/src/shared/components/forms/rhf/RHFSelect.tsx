import { useMemo, useState } from 'react'
import { Box, Select, MenuItem, FormControl, TextField, InputAdornment } from '@mui/material'
import SearchIcon from '@mui/icons-material/Search'
import { Controller, ControllerProps, FieldValues, useFormContext } from 'react-hook-form'
import { FieldLabel, FieldLabelProps } from 'src/components'

type ControllerRules = ControllerProps<FieldValues>['rules']

type FieldLabelCustomProps = Omit<FieldLabelProps, 'label' | 'helperText'>

export interface Option {
  label: string
  value: any
}

export type RHFSelectProps = {
  name: ControllerProps<FieldValues>['name']
  options: Option[]
  rules?: ControllerRules
  labelHelperText?: FieldLabelProps['helperText']
  labelProps?: FieldLabelCustomProps
  placeholder?: string
  label?: string
  disabled?: boolean
  helperText?: string
  error?: boolean
  searchable?: boolean
  searchPlaceholder?: string
}

export default function RHFSelect({
  name,
  options,
  rules,
  helperText,
  error: errorProp,
  label,
  labelHelperText,
  labelProps,
  placeholder,
  disabled,
  searchable,
  searchPlaceholder,
  ...rest
}: RHFSelectProps) {
  const { control } = useFormContext()

  const [open, setOpen] = useState(false)
  const [searchTerm, setSearchTerm] = useState('')

  const filteredOptions = useMemo(() => {
    if (!searchable || !searchTerm.trim()) {
      return options
    }

    const term = searchTerm.trim().toLowerCase()
    return options.filter((option) => option.label.toLowerCase().includes(term))
  }, [options, searchable, searchTerm])

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
              required={(rest as any).required}
              align="flex-start"
              {...labelProps}
            />
          ) : null}
          <FormControl
            variant="outlined"
            size="small"
            disabled={disabled}
            error={fieldState.invalid || errorProp}
          >
            <Select
              {...(rest as any)}
              {...field}
              labelId={label ? `${name}-label` : undefined}
              label={''}
              value={field.value ?? ''}
              displayEmpty
              open={searchable ? open : undefined}
              onOpen={
                searchable
                  ? () => {
                      setOpen(true)
                    }
                  : (rest as any).onOpen
              }
              onClose={
                searchable
                  ? (event) => {
                      setOpen(false)
                      setSearchTerm('')
                      ;(rest as any).onClose?.(event)
                    }
                  : (rest as any).onClose
              }
              MenuProps={{
                PaperProps: {
                  style: {
                    maxHeight: 300
                  }
                },
                MenuListProps: {
                  autoFocus: false
                },
                ...(((rest as any).MenuProps as any) ?? {})
              }}
            >
              {searchable && (
                <Box
                  sx={{
                    p: 1,
                    position: 'sticky',
                    top: 0,
                    bgcolor: 'background.paper',
                    zIndex: 1
                  }}
                >
                  <TextField
                    size="small"
                    placeholder={searchPlaceholder || 'Search...'}
                    fullWidth
                    value={searchTerm}
                    onChange={(e) => {
                      e.stopPropagation()
                      setSearchTerm(e.target.value)
                    }}
                    onClick={(e) => {
                      e.stopPropagation()
                      if (!open) {
                        setOpen(true)
                      }
                    }}
                    onKeyDown={(e) => {
                      e.stopPropagation()
                      // Prevent backspace from closing the dropdown
                      if (e.key === 'Backspace') {
                        ;(e.nativeEvent as any).stopImmediatePropagation?.()
                      }
                    }}
                    InputProps={{
                      startAdornment: (
                        <InputAdornment position="start">
                          <SearchIcon fontSize="small" />
                        </InputAdornment>
                      )
                    }}
                  />
                </Box>
              )}
              {placeholder && (
                <MenuItem value="" disabled>
                  <em>{placeholder}</em>
                </MenuItem>
              )}
              {searchable && filteredOptions.length === 0 ? (
                <MenuItem disabled>No matching options</MenuItem>
              ) : (
                (searchable ? filteredOptions : options).map((option) => (
                  <MenuItem key={String(option.value)} value={option.value}>
                    {option.label}
                  </MenuItem>
                ))
              )}
            </Select>
          </FormControl>
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
