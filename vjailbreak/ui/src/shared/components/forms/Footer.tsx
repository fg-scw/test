import { Button, CircularProgress, styled } from '@mui/material'
import { useThemeContext } from 'src/theme/ThemeContext'

interface FooterProps {
  cancelButtonLabel?: string
  submitButtonLabel?: string
  onClose: () => void
  onSubmit: () => void
  disableSubmit?: boolean
  submitting?: boolean
}

const StyledFooter = styled('div')(({ theme }) => ({
  display: 'flex',
  justifyItems: 'end',
  justifyContent: 'end',
  gap: theme.spacing(2),
  // marginTop: "auto",
  padding: theme.spacing(2),
  borderTop: `1px solid ${theme.palette.divider}`
}))

export default function Footer({
  cancelButtonLabel = 'Cancel',
  submitButtonLabel = 'Submit',
  onClose,
  onSubmit,
  submitting = false,
  disableSubmit = false
}: FooterProps) {
  const { mode } = useThemeContext()

  return (
    <StyledFooter>
      <Button
        type="button"
        variant={mode === 'dark' ? 'contained' : 'outlined'}
        color="secondary"
        onClick={onClose}
      >
        {cancelButtonLabel}
      </Button>
      <Button
        type="submit"
        variant="contained"
        color="primary"
        onClick={onSubmit}
        disabled={disableSubmit || submitting}
      >
        {submitting && <CircularProgress size={20} sx={{ marginRight: 2 }} />}
        {submitting ? 'Submitting' : submitButtonLabel}
      </Button>
    </StyledFooter>
  )
}
