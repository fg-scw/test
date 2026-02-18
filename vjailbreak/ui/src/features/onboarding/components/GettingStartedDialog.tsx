import { Button, Typography, Box, Dialog, Divider } from '@mui/material'
import { useNavigate } from 'react-router-dom'

export type GettingStartedDialogProps = {
  open: boolean
  onClose: () => void
  onDismiss: () => void
  showVddkStep: boolean
  showCredentialsStep: boolean
  vddkFirst: boolean
}

export default function GettingStartedDialog({
  open,
  onClose,
  onDismiss,
  showVddkStep,
  showCredentialsStep,
  vddkFirst
}: GettingStartedDialogProps) {
  const navigate = useNavigate()

  const goToCredentials = () => {
    onClose()
    navigate('/dashboard/credentials')
  }

  const goToVddkUpload = () => {
    onClose()
    navigate('/dashboard/global-settings')
  }

  const vddkStep = {
    key: 'vddk' as const,
    title: 'Upload VDDK library',
    body: 'Upload the VMware VDDK tar/tar.gz under Global Settings.',
    action: (
      <Button variant="contained" onClick={goToVddkUpload}>
        Go to VDDK Upload
      </Button>
    )
  }

  const credsStep = {
    key: 'creds' as const,
    title: 'Add credentials',
    body: 'Add PCD and VMware credentials from the Credentials page.',
    action: (
      <Button variant="contained" onClick={goToCredentials}>
        Go to Credentials
      </Button>
    )
  }

  const orderedSteps: Array<typeof vddkStep | typeof credsStep> = []

  if (vddkFirst) {
    if (showVddkStep) orderedSteps.push(vddkStep)
    if (showCredentialsStep) orderedSteps.push(credsStep)
  } else {
    if (showCredentialsStep) orderedSteps.push(credsStep)
    if (showVddkStep) orderedSteps.push(vddkStep)
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      PaperProps={{
        sx: {
          p: 2,
          width: 420,
          borderRadius: 2
        }
      }}
    >
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5, minWidth: 360 }}>
        <Typography variant="subtitle1" fontWeight={800}>
          Next steps
        </Typography>

        <Typography variant="body2" color="text.secondary">
          Complete the items below to start migrations.
        </Typography>

        <Divider />

        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
          {orderedSteps.map((step, idx) => (
            <Box key={step.key} sx={{ display: 'flex', flexDirection: 'column', gap: 0.75 }}>
              <Typography variant="subtitle2" fontWeight={700}>
                {idx + 1}) {step.title}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                {step.body}
              </Typography>
              <Box>{step.action}</Box>
            </Box>
          ))}
        </Box>

        <Divider />

        <Box sx={{ display: 'flex', gap: 1, justifyContent: 'flex-end' }}>
          <Button
            onClick={() => {
              onDismiss()
              onClose()
            }}
            color="inherit"
            size="small"
          >
            Don’t show again
          </Button>
          <Button onClick={onClose} variant="contained" size="small">
            Close
          </Button>
        </Box>
      </Box>
    </Dialog>
  )
}
