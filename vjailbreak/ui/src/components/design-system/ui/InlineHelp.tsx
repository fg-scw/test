import { Box, IconButton, Typography, BoxProps, Theme, useTheme } from '@mui/material'
import {
  CheckCircleOutline,
  Close,
  ErrorOutline,
  InfoOutlined,
  WarningAmberOutlined
} from '@mui/icons-material'
import { forwardRef, type ReactElement } from 'react'

export type InlineHelpTone = 'default' | 'positive' | 'critical' | 'warning'

export type InlineHelpVariant = 'contained' | 'outline'

export type InlineHelpIcon = 'none' | 'auto' | 'info' | 'success' | 'warning' | 'danger'

export interface InlineHelpProps extends BoxProps {
  children: React.ReactNode
  tone?: InlineHelpTone

  variant?: InlineHelpVariant
  icon?: InlineHelpIcon | ReactElement

  onClose?: () => void
  closeAriaLabel?: string
}

const getToneStyles = (tone: InlineHelpTone, variant: InlineHelpVariant, theme: Theme) => {
  switch (tone) {
    case 'positive': {
      const backgroundColor = variant === 'outline' ? 'transparent' : theme.palette.success.light
      return {
        color:
          variant === 'outline'
            ? theme.palette.success.main
            : theme.palette.getContrastText(theme.palette.success.light),
        backgroundColor,
        borderColor: theme.palette.success.main
      }
    }
    case 'critical': {
      const backgroundColor = variant === 'outline' ? 'transparent' : theme.palette.error.light
      return {
        color:
          variant === 'outline'
            ? theme.palette.error.main
            : theme.palette.getContrastText(theme.palette.error.light),
        backgroundColor,
        borderColor: theme.palette.error.main
      }
    }
    case 'warning': {
      const backgroundColor = variant === 'outline' ? 'transparent' : theme.palette.warning.light
      return {
        color:
          variant === 'outline'
            ? theme.palette.warning.main
            : theme.palette.getContrastText(theme.palette.warning.light),
        backgroundColor,
        borderColor: theme.palette.warning.main
      }
    }
    default:
      return {
        color: theme.palette.text.secondary,
        backgroundColor: variant === 'outline' ? 'transparent' : theme.palette.grey[100],
        borderColor: theme.palette.grey[300]
      }
  }
}

const getAutoIcon = (tone: InlineHelpTone) => {
  switch (tone) {
    case 'positive':
      return 'success'
    case 'critical':
      return 'danger'
    case 'warning':
      return 'warning'
    default:
      return 'info'
  }
}

const getIconNode = (icon: InlineHelpIcon, tone: InlineHelpTone) => {
  const resolved = icon === 'auto' ? getAutoIcon(tone) : icon

  switch (resolved) {
    case 'success':
      return <CheckCircleOutline fontSize="inherit" />
    case 'warning':
      return <WarningAmberOutlined fontSize="inherit" />
    case 'danger':
      return <ErrorOutline fontSize="inherit" />
    case 'info':
      return <InfoOutlined fontSize="inherit" />
    case 'none':
    default:
      return null
  }
}

const InlineHelp = forwardRef<HTMLDivElement, InlineHelpProps>(function InlineHelp(
  {
    children,
    tone = 'default',
    variant = 'outline',
    icon = 'none',
    onClose,
    closeAriaLabel = 'Close',
    sx,
    ...rest
  },
  ref
) {
  const theme = useTheme()
  const iconNode = typeof icon === 'string' ? getIconNode(icon, tone) : icon
  const hasLeadingIcon = Boolean(iconNode)

  const baseSx = {
    p: 1,
    borderRadius: 1,
    border: 1,
    fontSize: '0.875rem',
    ...getToneStyles(tone, variant, theme),
    display: 'flex',
    gap: 1,
    alignItems: 'center'
  }

  const mergedSx = Array.isArray(sx) ? [baseSx, ...sx] : [baseSx, sx]

  return (
    <Box ref={ref} sx={mergedSx} {...rest}>
      {hasLeadingIcon ? (
        <Box sx={{ fontSize: 18, lineHeight: 1, flexShrink: 0, display: 'flex', mt: '2px' }}>
          {iconNode}
        </Box>
      ) : null}
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Typography variant="body2" component="div">
          {children}
        </Typography>
      </Box>
      {onClose ? (
        <IconButton
          aria-label={closeAriaLabel}
          size="small"
          onClick={onClose}
          sx={{ mt: '-2px', mr: '-4px' }}
        >
          <Close fontSize="small" />
        </IconButton>
      ) : null}
    </Box>
  )
})

export default InlineHelp
