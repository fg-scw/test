import { Box, Drawer, DrawerProps, IconButton, Typography } from '@mui/material'
import CloseIcon from '@mui/icons-material/Close'
import { ReactNode, useRef, useState } from 'react'
import ConfirmationDialog from 'src/components/dialogs/ConfirmationDialog'

export interface DrawerShellProps extends DrawerProps {
  width?: number | string
  header?: ReactNode
  footer?: ReactNode
  'data-testid'?: string
  requireCloseConfirmation?: boolean
}

export default function DrawerShell({
  width = 820,
  header,
  footer,
  children,
  PaperProps,
  'data-testid': dataTestId = 'drawer-shell',
  requireCloseConfirmation = true,
  ...props
}: DrawerShellProps) {
  const [confirmOpen, setConfirmOpen] = useState(false)
  const pendingCloseRef = useRef<{ event: unknown; reason: unknown } | null>(null)

  const handleCloseConfirmed = async () => {
    const pending = pendingCloseRef.current
    pendingCloseRef.current = null
    setConfirmOpen(false)
    props.onClose?.(pending?.event as never, pending?.reason as never)
  }

  const handleCloseConfirmationDismiss = () => {
    pendingCloseRef.current = null
    setConfirmOpen(false)
  }

  const handleDrawerClose: DrawerProps['onClose'] = (event, reason) => {
    if (requireCloseConfirmation && (reason === 'backdropClick' || reason === 'escapeKeyDown')) {
      pendingCloseRef.current = { event, reason }
      setConfirmOpen(true)
      return
    }

    props.onClose?.(event, reason)
  }

  return (
    <>
      <Drawer
        anchor="right"
        {...props}
        onClose={handleDrawerClose}
        PaperProps={{
          sx: {
            display: 'grid',
            gridTemplateRows:
              `${header ? 'max-content' : ''} 1fr ${footer ? 'max-content' : ''}`.trim(),
            width,
            maxWidth: '95vw',
            borderLeft: (theme) => `1px solid ${theme.palette.divider}`,
            backgroundColor: (theme) => theme.palette.background.paper,
            ...(PaperProps?.sx ?? {})
          },
          ...PaperProps
        }}
        data-testid={dataTestId}
      >
        {header}
        <DrawerBody>{children}</DrawerBody>
        {footer}
      </Drawer>

      <ConfirmationDialog
        open={confirmOpen}
        onClose={handleCloseConfirmationDismiss}
        title="Discard Changes?"
        message="Are you sure you want to leave? Any unsaved changes will be lost."
        actionLabel="Leave"
        actionColor="warning"
        actionVariant="outlined"
        onConfirm={handleCloseConfirmed}
      />
    </>
  )
}

export interface DrawerHeaderProps {
  title: ReactNode
  subtitle?: ReactNode
  icon?: ReactNode
  actions?: ReactNode
  onClose?: () => void
  'data-testid'?: string
}

export function DrawerHeader({
  title,
  subtitle,
  icon,
  actions,
  onClose,
  'data-testid': dataTestId = 'drawer-header'
}: DrawerHeaderProps) {
  return (
    <Box
      data-testid={dataTestId}
      sx={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        px: 4,
        py: 3,
        borderBottom: (theme) => `1px solid ${theme.palette.divider}`
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
        {icon ? <Box sx={{ display: 'flex' }}>{icon}</Box> : null}
        <Box>
          <Typography variant="h5" component="h2">
            {title}
          </Typography>
          {subtitle ? (
            <Typography variant="body2" color="text.secondary">
              {subtitle}
            </Typography>
          ) : null}
        </Box>
      </Box>

      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        {actions}
        {onClose ? (
          <IconButton
            aria-label="Close drawer"
            onClick={onClose}
            data-testid={`${dataTestId}-close`}
          >
            <CloseIcon />
          </IconButton>
        ) : null}
      </Box>
    </Box>
  )
}

export interface DrawerBodyProps {
  children: ReactNode
  'data-testid'?: string
}

export function DrawerBody({
  children,
  'data-testid': dataTestId = 'drawer-body'
}: DrawerBodyProps) {
  return (
    <Box data-testid={dataTestId} sx={{ overflow: 'auto', px: 4, py: 4 }}>
      {children}
    </Box>
  )
}

export interface DrawerFooterProps {
  children: ReactNode
  'data-testid'?: string
}

export function DrawerFooter({
  children,
  'data-testid': dataTestId = 'drawer-footer'
}: DrawerFooterProps) {
  return (
    <Box
      data-testid={dataTestId}
      sx={{
        borderTop: (theme) => `1px solid ${theme.palette.divider}`,
        px: 4,
        py: 3,
        display: 'flex',
        justifyContent: 'flex-end',
        gap: 2
      }}
    >
      {children}
    </Box>
  )
}
