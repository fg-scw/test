import { useMemo } from 'react'
import { CircularProgress, styled, Typography, Box, Tooltip } from '@mui/material'
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline'
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline'
import HourglassBottomIcon from '@mui/icons-material/HourglassBottom'
import PauseCircleOutlineIcon from '@mui/icons-material/PauseCircleOutline'
import { Phase } from 'src/api/migrations/model'

const ProgressContainer = styled(Box)({
  display: 'flex',
  alignItems: 'center',
  height: '100%',
  textOverflow: 'ellipsis',
  whiteSpace: 'nowrap',
  maxWidth: '100%',
  overflow: 'hidden'
})

interface MigrationProgressProps {
  phase: Phase
  progressText: string
}

export default function MigrationProgress({ progressText, phase }: MigrationProgressProps) {
  const statusIcon = useMemo(() => {
    if (phase === Phase.Succeeded) {
      return <CheckCircleOutlineIcon style={{ color: 'green' }} />
    } else if (phase === Phase.AwaitingAdminCutOver) {
      return <PauseCircleOutlineIcon style={{ color: '#1976d2' }} />
    } else if (
      [
        Phase.Validating,
        Phase.AwaitingDataCopyStart,
        Phase.CopyingBlocks,
        Phase.CopyingChangedBlocks,
        Phase.ConvertingDisk,
        Phase.AwaitingCutOverStartTime
      ].includes(phase as Phase)
    ) {
      return <CircularProgress size={20} style={{ marginRight: 3 }} />
    } else if (phase === Phase.Failed || phase === Phase.ValidationFailed) {
      return <ErrorOutlineIcon style={{ color: 'red' }} />
    } else {
      return <HourglassBottomIcon style={{ color: 'grey' }} />
    }
  }, [phase])

  return (
    <>
      <ProgressContainer>
        {statusIcon}
        <Tooltip title={progressText} arrow>
          <Typography
            variant="body2"
            sx={{
              ml: 2,
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
              maxWidth: '100%'
            }}
          >
            {progressText}
          </Typography>
        </Tooltip>
      </ProgressContainer>
    </>
  )
}
