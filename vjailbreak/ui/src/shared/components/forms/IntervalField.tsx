import { Box, TextField } from '@mui/material'
import { useCallback, useEffect, useState } from 'react'

type IntervalFieldProps = {
  label: string
  name: string
  value: string
  helper?: string
  error?: string
  disabled?: boolean
  required?: boolean
  getErrorsUpdater?: any
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void
}

const IntervalField = ({
  label,
  name,
  value,
  helper,
  error,
  disabled,
  required,
  onChange,
  getErrorsUpdater
}: IntervalFieldProps) => {
  const [validationError, setValidationError] = useState<string | undefined>(undefined)

  const validate = useCallback((val: string): string | undefined => {
    const trimmedVal = val?.trim()
    if (!trimmedVal) return undefined

    // Allow composite formats like 1h30m, 5m30s, etc.
    const regex = /^(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s)?$/
    const match = trimmedVal.match(regex)

    if (
      !match ||
      match[0] === '' ||
      (match[1] === undefined && match[2] === undefined && match[3] === undefined)
    ) {
      return 'Use duration format like 5m, 1h30m, 5m30s (units: h,m,s).'
    }

    const hours = match[1] ? Number(match[1]) : 0
    const minutes = match[2] ? Number(match[2]) : 0
    const seconds = match[3] ? Number(match[3]) : 0

    // Convert total duration to minutes
    const totalMinutes = hours * 60 + minutes + seconds / 60

    if (isNaN(totalMinutes) || totalMinutes < 5) {
      return 'Interval must be at least 5 minutes'
    }

    return undefined
  }, [])

  const updateValidationError = useCallback(
    (newValue: string) => {
      const newValidationError = validate(newValue)
      setValidationError(newValidationError)
      if (getErrorsUpdater) {
        getErrorsUpdater(name)(newValidationError || '')
      }
    },
    [validate, getErrorsUpdater, name]
  )

  const handleInputChange = useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      const newValue = event.target.value
      updateValidationError(newValue)
      onChange?.(event)
    },
    [updateValidationError, onChange]
  )

  useEffect(() => {
    updateValidationError(value)
  }, [value, updateValidationError])

  return (
    <Box display="flex" flexDirection="column" gap={0.5}>
      {/* <Typography variant="body2" fontWeight={500}>
        {label} {required && '*'}
      </Typography> */}
      <TextField
        fullWidth
        size="small"
        name={String(name)}
        placeholder={`${label}${required ? ' *' : ''}`}
        value={value}
        onChange={handleInputChange}
        disabled={disabled}
        required={required}
        error={!!error || !!validationError?.trim()}
        helperText={error || helper || validationError || 'e.g. 5m, 1h30m, 5m30s (units: h,m,s)'}
      />
    </Box>
  )
}

export default IntervalField
