import { DialogContent, DialogTitle, DialogActions, Button, Box, Typography } from '@mui/material'
import type { BMConfig } from 'src/api/bmconfig/model'

interface MaasConfigDetailsModalProps {
  open: boolean
  config: BMConfig | null
  onClose: () => void
}

export default function MaasConfigDetailsModal({ config, onClose }: MaasConfigDetailsModalProps) {
  if (!config) return null

  const spec = config.spec || ({} as any)

  return (
    <>
      <DialogTitle>MAAS Configuration Details</DialogTitle>
      <DialogContent dividers>
        <Box display="flex" flexDirection="column" gap={1}>
          <Typography variant="body2">
            <strong>Name:</strong> {config.metadata?.name}
          </Typography>
          {spec.apiUrl && (
            <Typography variant="body2">
              <strong>API URL:</strong> {spec.apiUrl}
            </Typography>
          )}
          {spec.os && (
            <Typography variant="body2">
              <strong>OS:</strong> {spec.os}
            </Typography>
          )}
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </>
  )
}
