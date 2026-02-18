import CloudUploadOutlinedIcon from '@mui/icons-material/CloudUploadOutlined'
import { Box, Paper, Typography, styled } from '@mui/material'
import {
  useCallback,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
  type DragEvent,
  type ReactNode
} from 'react'

const DropzoneContainer = styled(Paper)(({ theme }) => ({
  padding: theme.spacing(3),
  border: `2px dashed ${theme.palette.divider}`,
  borderRadius: theme.shape.borderRadius,
  textAlign: 'center',
  cursor: 'pointer',
  transition: 'all 0.2s ease',
  backgroundColor: theme.palette.background.paper,
  '&:hover': {
    borderColor: theme.palette.primary.main,
    backgroundColor: theme.palette.action.hover
  },
  '&.dragging': {
    borderColor: theme.palette.primary.main,
    backgroundColor: theme.palette.action.selected
  }
}))

export type FileDropzoneProps = {
  id?: string
  accept?: string
  disabled?: boolean
  file?: File | null
  placeholder?: string
  helperText?: string
  caption?: string
  icon?: ReactNode
  onFileSelected: (file: File | null) => void
  'data-testid'?: string
}

export default function FileDropzone({
  id,
  accept,
  disabled = false,
  file,
  placeholder = 'Drag and drop a file here',
  helperText = 'or click to browse',
  caption,
  icon,
  onFileSelected,
  'data-testid': dataTestId = 'file-dropzone'
}: FileDropzoneProps) {
  const inputRef = useRef<HTMLInputElement | null>(null)
  const [isDragging, setIsDragging] = useState(false)

  const inputId = useMemo(
    () => id ?? `file-dropzone-input-${Math.random().toString(36).slice(2)}`,
    [id]
  )

  const handleFile = useCallback(
    (next: File | null) => {
      if (disabled) return
      onFileSelected(next)
    },
    [disabled, onFileSelected]
  )

  const handleClick = useCallback(() => {
    if (disabled) return
    inputRef.current?.click()
  }, [disabled])

  const handleInputChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      const nextFile = event.target.files?.[0] || null
      handleFile(nextFile)
      // Reset so the same file can be selected again
      event.target.value = ''
    },
    [handleFile]
  )

  const handleDragOver = useCallback(
    (event: DragEvent) => {
      if (disabled) return
      event.preventDefault()
      setIsDragging(true)
    },
    [disabled]
  )

  const handleDragLeave = useCallback(
    (event: DragEvent) => {
      if (disabled) return
      event.preventDefault()
      setIsDragging(false)
    },
    [disabled]
  )

  const handleDrop = useCallback(
    (event: DragEvent) => {
      if (disabled) return
      event.preventDefault()
      setIsDragging(false)
      const nextFile = event.dataTransfer.files?.[0] || null
      handleFile(nextFile)
    },
    [disabled, handleFile]
  )

  return (
    <Box>
      <DropzoneContainer
        className={isDragging ? 'dragging' : ''}
        onClick={handleClick}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        elevation={0}
        data-testid={dataTestId}
        sx={{
          opacity: disabled ? 0.6 : 1
        }}
      >
        {icon ?? <CloudUploadOutlinedIcon sx={{ fontSize: 40, color: 'text.secondary', mb: 1 }} />}
        <Typography variant="body2" fontWeight={600} gutterBottom>
          {file?.name || placeholder}
        </Typography>
        <Typography variant="body2" color="text.secondary">
          {helperText}
        </Typography>
        {caption ? (
          <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: 'block' }}>
            {caption}
          </Typography>
        ) : null}
      </DropzoneContainer>

      <input
        id={inputId}
        ref={inputRef}
        type="file"
        accept={accept}
        onChange={handleInputChange}
        disabled={disabled}
        style={{ display: 'none' }}
      />
    </Box>
  )
}
