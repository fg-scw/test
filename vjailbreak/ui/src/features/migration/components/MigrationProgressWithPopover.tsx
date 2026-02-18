import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Stepper,
  Step,
  StepLabel,
  CircularProgress,
  styled,
  Popover,
  Typography,
  Box,
  StepIconProps
} from '@mui/material'
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline'
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline'
import HourglassBottomIcon from '@mui/icons-material/HourglassBottom'
import PauseCircleOutlineIcon from '@mui/icons-material/PauseCircleOutline'
import { Phase, Condition } from 'src/api/migrations/model'

enum Status {
  Pending = 'Pending',
  Completed = 'Completed',
  InProgress = 'InProgress',
  Failed = 'Failed'
}

enum StatusType {
  Validated = 'Validated',
  DataCopy = 'DataCopy',
  Migrated = 'Migrated'
}

interface StatusStep {
  type: StatusType
  status: Status
  message: string
  reason?: string
  lastTransitionTime?: string
}

const defaultSteps: Record<StatusType, StatusStep> = {
  [StatusType.Validated]: {
    type: StatusType.Validated,
    status: Status.Pending,
    message: 'Validating'
  },
  [StatusType.DataCopy]: {
    type: StatusType.DataCopy,
    status: Status.Pending,
    message: 'Copying Data'
  },
  [StatusType.Migrated]: {
    type: StatusType.Migrated,
    status: Status.Pending,
    message: 'Migrating'
  }
}

const StepperContainer = styled('div')({
  width: '100%',
  padding: '4px',
  margin: '12px'
})

const ProgressContainer = styled(Box)({
  display: 'flex',
  alignItems: 'center',
  height: '100%',
  cursor: 'pointer'
})

interface MigrationProgressWithPopoverProps {
  phase: Phase | undefined
  conditions: Condition[]
  progressText: string
}

export default function MigrationProgressWithPopover({
  phase,
  conditions,
  progressText
}: MigrationProgressWithPopoverProps) {
  const [steps, setSteps] = useState<StatusStep[]>(Object.values(defaultSteps))
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null)

  const mapStatus = (status: string): Status => {
    if (status === 'True') return Status.Completed
    if (status === 'Unknown') return Status.InProgress
    if (status === 'False') return Status.Failed
    return Status.Pending
  }

  useEffect(() => {
    const updatedSteps = conditions.reduce(
      (acc, currentStep) => {
        const { type, status } = currentStep as unknown as { type: StatusType; status: string }

        acc[type] = {
          ...(currentStep as unknown as StatusStep),
          status: mapStatus(status)
        }

        return acc
      },
      JSON.parse(JSON.stringify(defaultSteps)) as Record<StatusType, StatusStep>
    )

    setSteps(Object.values(updatedSteps))
  }, [conditions])

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

  const activeStepIndex: number = useMemo(() => {
    return steps.findIndex(
      (step) => step.status === Status.InProgress || step.status === Status.Failed
    )
  }, [steps])

  const activeStep: StatusStep = useMemo(
    () => steps[Math.max(activeStepIndex, 0)] ?? steps[0],
    [steps, activeStepIndex]
  )

  const handlePopoverOpen = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget)
  }

  const handlePopoverClose = () => {
    setAnchorEl(null)
  }

  return (
    <ProgressContainer onMouseEnter={handlePopoverOpen} onMouseLeave={handlePopoverClose}>
      {statusIcon}
      <Typography variant="body2" sx={{ ml: 2 }}>
        {progressText}
      </Typography>

      <Popover
        id="mouse-over-popover"
        sx={{ pointerEvents: 'none' }}
        open={Boolean(anchorEl)}
        anchorEl={anchorEl}
        anchorOrigin={{
          vertical: 'top',
          horizontal: 'left'
        }}
        transformOrigin={{
          vertical: 'top',
          horizontal: 'left'
        }}
        onClose={handlePopoverClose}
        disableRestoreFocus
      >
        <StepperComponent activeStepIndex={activeStepIndex} activeStep={activeStep} steps={steps} />
      </Popover>
    </ProgressContainer>
  )
}

interface StepperComponentProps {
  activeStepIndex: number
  activeStep: StatusStep
  steps: StatusStep[]
}

const StepperComponent = ({ activeStepIndex, activeStep, steps }: StepperComponentProps) => {
  const getStepIcon = useCallback((status: Status) => {
    const StepIcon = (_props: StepIconProps) => {
      switch (status) {
        case Status.Completed:
          return <CheckCircleOutlineIcon style={{ color: 'green' }} />
        case Status.Pending:
          return <HourglassBottomIcon style={{ color: 'grey' }} />
        case Status.InProgress:
          return <CircularProgress size={20} sx={{ color: 'green' }} />
        case Status.Failed:
          return <ErrorOutlineIcon style={{ color: 'red' }} />
        default:
          return null
      }
    }
    return StepIcon
  }, [])

  const lastUpdated = activeStep?.lastTransitionTime
    ? new Date(String(activeStep?.lastTransitionTime)).toLocaleTimeString('en-US')
    : null

  const diffInMinutes = useMemo(() => {
    if (!activeStep?.lastTransitionTime) return null
    const localDate: Date = new Date()
    const utcTime: Date = new Date(String(activeStep?.lastTransitionTime))
    const timeDifferenceInMs: number = localDate.getTime() - utcTime.getTime()
    return Math.floor(timeDifferenceInMs / 1000 / 60)
  }, [activeStep?.lastTransitionTime])

  return (
    <StepperContainer>
      {lastUpdated && diffInMinutes !== null && (
        <Typography>{`Last Updated: ${lastUpdated} (${diffInMinutes} minutes ago)`}</Typography>
      )}
      <Stepper activeStep={activeStepIndex} orientation="vertical">
        {steps.map((step, index) => (
          <Step
            key={`${index}+${String(step.type)}`}
            completed={[Status.Completed, Status.Failed].includes(step.status)}
          >
            <StepLabel StepIconComponent={getStepIcon(step.status)}>
              <Typography
                sx={{
                  color: step.status === Status.Failed ? 'red' : 'default'
                }}
              >
                {step.message}
              </Typography>
            </StepLabel>
          </Step>
        ))}
      </Stepper>
    </StepperContainer>
  )
}
