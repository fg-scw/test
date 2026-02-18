import {
  Autocomplete,
  Box,
  Button,
  CircularProgress,
  FormControl,
  FormHelperText,
  TextField
} from '@mui/material'
import AddIcon from '@mui/icons-material/Add'
import { useState } from 'react'
import { FieldLabel } from 'src/components'

interface CredentialOption {
  label: string
  value: string
  metadata: {
    name: string
    namespace?: string
  }
  status?: {
    validationStatus?: string
    validationMessage?: string
  }
}

interface CredentialSelectorProps {
  label?: string
  placeholder?: string
  options: CredentialOption[]
  value: string | null
  onChange: (value: string | null) => void
  onAddNew?: () => void
  loading: boolean
  error?: string
  emptyMessage?: string
  size?: 'small' | 'medium'
  showAddNewButton?: boolean
}

export default function CredentialSelector({
  label,
  placeholder,
  options,
  value,
  onChange,
  onAddNew,
  loading,
  error,
  size = 'small',
  emptyMessage = 'No credentials found',
  showAddNewButton = false
}: CredentialSelectorProps) {
  const [inputValue, setInputValue] = useState('')

  const selectedOption = options.find((option) => option.value === value) || null

  return (
    <FormControl fullWidth error={!!error}>
      {label ? (
        <Box sx={{ mb: 0.5 }}>
          <FieldLabel label={label} align="flex-start" />
        </Box>
      ) : null}
      <Box sx={{ display: 'flex', gap: 1 }}>
        <Autocomplete
          fullWidth
          options={options}
          loading={loading}
          size={size}
          value={selectedOption}
          inputValue={inputValue}
          onInputChange={(_, newInputValue) => {
            setInputValue(newInputValue)
          }}
          onChange={(_, newValue) => {
            onChange(newValue?.value || null)
          }}
          getOptionLabel={(option) => option.label}
          noOptionsText={emptyMessage}
          renderInput={(params) => (
            <TextField
              {...params}
              label={label}
              placeholder={placeholder}
              variant="outlined"
              size={size}
              InputProps={{
                ...params.InputProps,
                endAdornment: (
                  <>
                    {loading ? <CircularProgress color="inherit" size={20} /> : null}
                    {params.InputProps.endAdornment}
                  </>
                )
              }}
            />
          )}
        />
        {showAddNewButton && onAddNew && (
          <Button
            color="primary"
            onClick={onAddNew}
            startIcon={<AddIcon />}
            sx={{ minWidth: '120px' }}
          >
            Add New
          </Button>
        )}
      </Box>
      {!!error && (
        <Box sx={{ mt: 1 }}>
          <FormHelperText error>{error}</FormHelperText>
        </Box>
      )}
    </FormControl>
  )
}
