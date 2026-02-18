import { Button, ButtonProps, CircularProgress } from '@mui/material'
import { forwardRef } from 'react'

type ButtonTone = 'primary' | 'secondary' | 'danger'

const toneDefaults: Record<
  ButtonTone,
  { variant: ButtonProps['variant']; color: ButtonProps['color'] }
> = {
  primary: { variant: 'contained', color: 'primary' },
  secondary: { variant: 'outlined', color: 'secondary' },
  danger: { variant: 'contained', color: 'error' }
}

export interface ActionButtonProps extends ButtonProps {
  tone?: ButtonTone
  loading?: boolean
  'data-testid'?: string
}

const ActionButton = forwardRef<HTMLButtonElement, ActionButtonProps>(function ActionButton(
  {
    tone = 'primary',
    loading = false,
    disabled,
    children,
    'data-testid': dataTestId = 'action-button',
    variant,
    color,
    startIcon,
    ...rest
  },
  ref
) {
  const toneConfig = toneDefaults[tone]

  const computedStartIcon = loading ? (
    <CircularProgress size={16} data-testid={`${dataTestId}-spinner`} />
  ) : (
    startIcon
  )

  return (
    <Button
      ref={ref}
      variant={variant ?? toneConfig.variant}
      color={color ?? toneConfig.color}
      startIcon={computedStartIcon}
      disabled={disabled || loading}
      data-testid={dataTestId}
      {...rest}
    >
      {children}
    </Button>
  )
})

export default ActionButton
