import { Box, BoxProps, CircularProgress } from '@mui/material'
import { ReactNode } from 'react'

import InlineHelp, { InlineHelpTone } from './InlineHelp'
import Row from './Row'

export type OperationStatusLayout = 'row' | 'text'

export interface OperationStatusProps extends BoxProps {
  loading?: boolean
  loadingMessage?: ReactNode
  loadingTone?: InlineHelpTone
  loadingLayout?: OperationStatusLayout

  success?: boolean
  successMessage?: ReactNode
  successTone?: InlineHelpTone
  successIcon?: ReactNode
  successLayout?: OperationStatusLayout

  error?: ReactNode | null
  errorTone?: InlineHelpTone
}

export default function OperationStatus({
  loading,
  loadingMessage,
  loadingTone = 'warning',
  loadingLayout = 'row',
  success,
  successMessage,
  successTone = 'positive',
  successIcon,
  successLayout = successIcon ? 'row' : 'text',
  error,
  errorTone = 'critical',
  ...boxProps
}: OperationStatusProps) {
  return (
    <Box {...boxProps}>
      {loading && (
        <InlineHelp tone={loadingTone}>
          {loadingLayout === 'row' ? (
            <Row gap={1}>
              <CircularProgress size={16} />
              <span>{loadingMessage}</span>
            </Row>
          ) : (
            loadingMessage
          )}
        </InlineHelp>
      )}

      {success && (
        <InlineHelp tone={successTone}>
          {successLayout === 'row' ? (
            <Row gap={1}>
              {successIcon}
              <span>{successMessage}</span>
            </Row>
          ) : (
            successMessage
          )}
        </InlineHelp>
      )}

      {error ? <InlineHelp tone={errorTone}>{error}</InlineHelp> : null}
    </Box>
  )
}
