import React, { useState } from 'react'
import {
  IconButton,
  Tooltip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Button,
  Alert,
  CircularProgress
} from '@mui/material'
import PlayArrowIcon from '@mui/icons-material/PlayArrow'
import { triggerAdminCutover } from '../../api/migrations'

interface TriggerAdminCutoverButtonProps {
  migrationName: string
  namespace?: string
  onSuccess?: () => void
  onError?: (error: string) => void
}

export const TriggerAdminCutoverButton: React.FC<TriggerAdminCutoverButtonProps> = ({
  migrationName,
  namespace,
  onSuccess,
  onError
}) => {
  const [open, setOpen] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)

  const handleTriggerCutover = async () => {
    setError(null)
    setIsLoading(true)

    try {
      const result = await triggerAdminCutover(namespace || 'migration-system', migrationName)

      if (result.success) {
        setOpen(false)
        onSuccess?.()
      } else {
        setError(result.message || 'Failed to trigger cutover')
        onError?.(result.message || 'Failed to trigger cutover')
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred'
      setError(errorMessage)
      onError?.(errorMessage)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <>
      <Tooltip title="Trigger Admin Cutover">
        <IconButton
          onClick={(e) => {
            e.stopPropagation()
            setError(null) // Clear any previous errors
            setOpen(true)
          }}
          size="small"
          disabled={isLoading}
          sx={{
            cursor: 'pointer',
            color: 'primary.main'
          }}
        >
          {isLoading ? <CircularProgress size={16} /> : <PlayArrowIcon />}
        </IconButton>
      </Tooltip>

      <Dialog open={open} onClose={() => setOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Trigger Admin Cutover</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Are you sure you want to trigger the admin cutover for migration "{migrationName}"? This
            will start the cutover process and cannot be undone.
          </DialogContentText>

          {error && (
            <Alert severity="error" sx={{ mt: 2 }}>
              {error}
            </Alert>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpen(false)} disabled={isLoading}>
            Cancel
          </Button>
          <Button
            onClick={handleTriggerCutover}
            variant="contained"
            color="primary"
            disabled={isLoading}
          >
            Trigger Cutover
          </Button>
        </DialogActions>
      </Dialog>
    </>
  )
}
