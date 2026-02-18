import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogContentText,
  DialogActions,
  Button,
  CircularProgress,
  Alert,
  Box,
  Typography,
  Tooltip,
  IconButton
} from '@mui/material'
import { useState } from 'react'
import CloseIcon from '@mui/icons-material/Close'

export interface ConfirmationDialogProps {
  // Dialog Control
  open: boolean
  onClose: () => void

  // Content
  title: string
  message: string | React.ReactNode
  icon?: React.ReactNode

  // Items being acted upon
  items?: Array<{
    id: string
    name: string
    [key: string]: unknown
  }>
  maxDisplayItems?: number

  actionLabel: string
  actionColor?: 'error' | 'warning' | 'info' | 'primary' | 'success'
  actionVariant?: 'text' | 'outlined' | 'contained'
  cancelLabel?: string

  onConfirm: () => Promise<void>

  additionalContent?: React.ReactNode

  errorMessage?: string | null
  onErrorChange?: (error: string | null) => void
  customErrorMessage?: (error: Error | string) => string
}

export default function ConfirmationDialog({
  open,
  onClose,
  title,
  message,
  icon,
  items = [],
  maxDisplayItems = 3,
  actionLabel,
  actionColor = 'primary',
  actionVariant = 'contained',
  cancelLabel = 'Cancel',
  onConfirm,
  additionalContent,
  errorMessage,
  onErrorChange,
  customErrorMessage
}: ConfirmationDialogProps) {
  const [isProcessing, setIsProcessing] = useState(false)
  const [internalError, setInternalError] = useState<string | null>(null)

  const displayedItems = items.slice(0, maxDisplayItems)
  const remainingCount = items.length - maxDisplayItems

  // Use either external or internal error state
  const error = errorMessage ?? internalError

  // Update error handling to use custom error message if provided
  const setError = (newError: string | null) => {
    if (onErrorChange) {
      onErrorChange(newError)
    } else {
      setInternalError(newError)
    }
  }

  const handleConfirm = async () => {
    setIsProcessing(true)
    setError(null)

    try {
      await onConfirm()
      onClose()
    } catch (error) {
      console.error('Action failed:', error)
      const errorMessage = customErrorMessage
        ? customErrorMessage(error as Error | string)
        : error instanceof Error
          ? error.message
          : 'Operation failed'

      setError(errorMessage)
    } finally {
      setIsProcessing(false)
    }
  }

  const handleClose = () => {
    if (!isProcessing) {
      setError(null)
      onClose()
    }
  }

  return (
    <Dialog
      open={open}
      onClose={handleClose}
      PaperProps={{
        sx: {
          width: '100%',
          maxWidth: '500px',
          m: 4
        }
      }}
    >
      <DialogTitle
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 1,
          pr: 6 // Space for close button
        }}
      >
        {icon}
        {title}
        <IconButton
          aria-label="close"
          onClick={handleClose}
          disabled={isProcessing}
          sx={{
            position: 'absolute',
            right: 8,
            top: 8
          }}
        >
          <CloseIcon />
        </IconButton>
      </DialogTitle>

      <DialogContent sx={{ px: 3, py: 2.5 }}>
        <DialogContentText>
          {message}
          {items.length > 1 && (
            <Box sx={{ mt: 1 }}>
              {displayedItems.map((item) => (
                <div key={item.id}>
                  • <strong>{item.name}</strong>
                </div>
              ))}
              {remainingCount > 0 && (
                <Tooltip
                  title={items
                    .slice(maxDisplayItems)
                    .map((item) => item.name)
                    .join('\n')}
                >
                  <Typography variant="body2">
                    and <strong>{remainingCount}</strong> more...
                  </Typography>
                </Tooltip>
              )}
            </Box>
          )}
        </DialogContentText>

        {additionalContent}

        {error && (
          <Alert severity="error" sx={{ mt: 2 }}>
            {error}
          </Alert>
        )}
      </DialogContent>

      <DialogActions sx={{ px: 3, pb: 2 }}>
        <Button onClick={handleClose} disabled={isProcessing}>
          {cancelLabel}
        </Button>
        <Button
          onClick={handleConfirm}
          color={actionColor}
          variant={actionVariant}
          disabled={isProcessing}
          sx={{ minWidth: 100 }}
        >
          {isProcessing && (
            <CircularProgress size={20} sx={{ marginRight: 2 }} color={actionColor} />
          )}
          {actionLabel}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
