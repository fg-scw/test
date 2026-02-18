import { useEffect, useState } from 'react'
import { useFormContext, Controller } from 'react-hook-form'
import { Box, FormControl, FormHelperText } from '@mui/material'
import { FieldLabel, ActionButton } from 'src/components'
import TextField from '../TextField'
import { parseRCFile, ParseRCFileResult } from 'src/utils/openstackRCFileParser'

export interface RHFOpenstackRCFileFieldProps {
  name?: string
  onParsed?: (values: Record<string, string>) => void
  onError?: (error: string) => void
  labelHelperText?: string
  required?: boolean
  size?: 'small' | 'medium'
  externalError?: string
}

/**
 * RHF-aware component for uploading and parsing OpenStack RC files.
 * Integrates with react-hook-form and automatically parses the uploaded file.
 * Uses custom UI matching the previous design with a read-only text field and button.
 */
export default function RHFOpenstackRCFileField({
  name = 'rcFile',
  onParsed,
  onError,
  labelHelperText = 'Upload the RC file exported from your OpenStack environment.',
  required = true,
  size = 'small',
  externalError
}: RHFOpenstackRCFileFieldProps) {
  const { control, watch, setError, clearErrors } = useFormContext()
  const [parsingError, setParsingError] = useState<string | null>(null)
  const [fileInputKey, setFileInputKey] = useState(0)
  const file = watch(name) as File | undefined

  useEffect(() => {
    if (!file) {
      setParsingError(null)
      clearErrors(name)
      return
    }

    let mounted = true

    parseRCFile(file)
      .then((result: ParseRCFileResult) => {
        if (!mounted) return

        if (result.success && result.fields) {
          setParsingError(null)
          clearErrors(name)
          onParsed?.(result.fields)
        } else {
          const errorMessage = result.error || 'Failed to parse RC file'
          setParsingError(errorMessage)
          setError(name, {
            type: 'manual',
            message: errorMessage
          })
          onError?.(errorMessage)
        }
      })
      .catch((error) => {
        if (!mounted) return
        const errorMessage = error instanceof Error ? error.message : 'Failed to parse RC file'
        setParsingError(errorMessage)
        setError(name, {
          type: 'manual',
          message: errorMessage
        })
        onError?.(errorMessage)
      })

    return () => {
      mounted = false
    }
  }, [file, name, setError, clearErrors, onParsed, onError])

  const fileName = file?.name || ''
  const displayError = parsingError || externalError
  const hasError = !!displayError

  return (
    <Controller
      name={name}
      control={control}
      rules={
        required
          ? {
              required: 'OpenStack RC file is required'
            }
          : undefined
      }
      render={({ field, fieldState }) => (
        <FormControl error={hasError || fieldState.invalid} fullWidth>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            <FieldLabel
              label="OpenStack RC file"
              required={required}
              align="flex-start"
              helperText={labelHelperText}
            />
            <Box
              sx={{
                display: 'grid',
                gridTemplateColumns: 'minmax(0, 1fr) auto',
                gridGap: (theme) => theme.spacing(2)
              }}
            >
              <TextField
                value={fileName}
                placeholder="No file selected"
                InputProps={{ readOnly: true }}
                size={size}
                required={required}
                error={hasError || fieldState.invalid}
              />
              <ActionButton
                tone="primary"
                component="label"
                size={size === 'small' ? 'small' : 'medium'}
              >
                Choose file
                <input
                  type="file"
                  hidden
                  onChange={(e) => {
                    const selectedFile = e.target.files?.[0]
                    if (selectedFile) {
                      field.onChange(selectedFile)
                      // Reset input so same file can be selected again
                      e.target.value = ''
                      setFileInputKey((prev) => prev + 1)
                    }
                  }}
                  key={fileInputKey}
                />
              </ActionButton>
            </Box>
          </Box>
          <FormHelperText error={hasError || fieldState.invalid}>
            {displayError || fieldState.error?.message || ' '}
          </FormHelperText>
        </FormControl>
      )}
    />
  )
}
