import { useState, useEffect } from 'react'
import { useQuery, useMutation } from '@tanstack/react-query'

import { UpgradeResponse, UpgradeProgressResponse } from 'src/api/version/model'
import Dialog from '@mui/material/Dialog'
import DialogTitle from '@mui/material/DialogTitle'
import Tooltip from '@mui/material/Tooltip'
import DialogContent from '@mui/material/DialogContent'
import DialogActions from '@mui/material/DialogActions'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import Select from '@mui/material/Select'
import MenuItem from '@mui/material/MenuItem'
import Alert from '@mui/material/Alert'
import CircularProgress from '@mui/material/CircularProgress'
import CheckCircleIcon from '@mui/icons-material/CheckCircle'
import { useTheme } from '@mui/material/styles'
import React from 'react'
import { ActionButton } from 'src/components'
import {
  cleanupApiCall,
  getAvailableTags,
  getUpgradeProgress,
  initiateUpgrade
} from 'src/api/version'

const getUIStatusMessage = (status: string | undefined): string => {
  switch (status) {
    case 'pending':
      return 'Pending'
    case 'in_progress':
    case 'deploying':
      return 'Upgrading'
    case 'verifying_stability':
      return 'Waiting for services to be ready'
    case 'rolling_back':
      return 'Rolling back'
    case 'completed':
      return 'Upgrade completed'
    case 'rolled_back':
      return 'Rolled back'
    case 'failed':
      return 'Upgrade failed'
    case 'rollback_failed':
      return 'Rollback failed'
    default:
      return 'Processing...'
  }
}

export const UpgradeModal = ({ show, onClose }) => {
  const [selectedVersion, setSelectedVersion] = useState('')
  const [errorMsg, setErrorMsg] = useState('')
  const [successMsg, setSuccessMsg] = useState('')
  const [upgradeInProgress, setUpgradeInProgress] = useState(false)
  const [cleanUpInProgress, setCleanUpInProgress] = useState(false)
  const [cleanupCompleted, setCleanupCompleted] = useState(false)
  const [progressData, setProgressData] = useState<UpgradeProgressResponse | null>(null)
  const theme = useTheme()

  const { data: updates, isLoading: areVersionsLoading } = useQuery({
    queryKey: ['availableTags'],
    queryFn: getAvailableTags,
    enabled: show
  })

  const cleanupMutation = useMutation({
    mutationFn: cleanupApiCall,
    onSuccess: (data) => {
      setCleanUpInProgress(false)
      if (data.success) {
        setCleanupCompleted(true)
        setSuccessMsg('Cleanup completed successfully')
        setErrorMsg('')
      } else {
        setErrorMsg(data.message || 'Cleanup failed')
        setSuccessMsg('')
      }
    },
    onError: (error: Error) => {
      setCleanUpInProgress(false)
      setErrorMsg(`Cleanup failed: ${error.message}`)
      setSuccessMsg('')
    }
  })

  const handleCleanup = () => {
    setCleanUpInProgress(true)
    setErrorMsg('')
    setSuccessMsg('')
    cleanupMutation.mutate()
  }

  const upgradeMutation = useMutation<UpgradeResponse, Error, void>({
    mutationFn: () => initiateUpgrade(selectedVersion, true),
    onSuccess: (data) => {
      if (data.upgradeStarted) {
        setUpgradeInProgress(true)
        setErrorMsg('')
      } else {
        setErrorMsg('Failed to start upgrade. Please try again.')
      }
    },
    onError: (error) => {
      setErrorMsg(`An error occurred: ${error.message}`)
      setSuccessMsg('')
    }
  })

  useEffect(() => {
    if (!upgradeInProgress) return

    const interval = setInterval(async () => {
      try {
        const progress = await getUpgradeProgress()
        setProgressData(progress)

        if (progress.status === 'deploying' || progress.status === 'in_progress' || progress.status === 'verifying_stability') {
          setSuccessMsg('')
          setErrorMsg('')
        } else if (progress.status === 'completed') {
          setUpgradeInProgress(false)
          setSuccessMsg('Upgrade completed successfully')
          clearInterval(interval)

          setTimeout(() => {
            sessionStorage.setItem('showUpgradeSuccess', 'true')
            sessionStorage.setItem('upgradedVersion', selectedVersion)
            onClose()
            window.location.href = '/dashboard/migrations'
          }, 3000)
        } else if (
          progress.status === 'failed' ||
          progress.status === 'rolled_back' ||
          progress.status === 'rollback_failed'
        ) {
          setUpgradeInProgress(false)
          setErrorMsg(progress.status === 'rolled_back' ? 'Upgrade failed: Rolled back to previous version' : 'Upgrade failed')
          clearInterval(interval)

          setTimeout(() => {
            window.location.reload()
          }, 2000)
        }
      } catch (err) {
        setUpgradeInProgress(false)
        setErrorMsg('Failed to fetch upgrade progress.')
        clearInterval(interval)
      }
    }, 3000)

    return () => clearInterval(interval)
  }, [upgradeInProgress, onClose, selectedVersion])

  if (!show) return null

  return (
    <React.Fragment>
      <Dialog open={show} onClose={upgradeInProgress || cleanUpInProgress ? undefined : onClose} maxWidth="xs" fullWidth>
        <DialogTitle>Upgrade vJailbreak</DialogTitle>
        <DialogContent>
          <Box mb={2}>
            <Select
              fullWidth
              value={selectedVersion}
              onChange={(e) => setSelectedVersion(e.target.value)}
              disabled={areVersionsLoading || upgradeMutation.isPending || upgradeInProgress || cleanUpInProgress}
              displayEmpty
              size="small"
            >
              <MenuItem value="">
                {areVersionsLoading ? 'Loading versions...' : 'Select a version...'}
              </MenuItem>
              {Array.isArray(updates?.updates) &&
                updates.updates.map((update) => (
                  <MenuItem key={update.version} value={update.version}>
                    {update.version}
                  </MenuItem>
                ))}
            </Select>
          </Box>

          {/* Pre-upgrade checklist info */}
          <Box
            mb={2}
            p={2}
            sx={{
              background: theme.palette.background.paper,
              border: `1px solid ${theme.palette.divider}`,
              borderRadius: 1,
              color: theme.palette.text.primary
            }}
          >
            <Typography variant="subtitle1" color="primary.main" fontWeight={600} gutterBottom>
              Pre-Upgrade Checklist
            </Typography>
            <Typography variant="body2" mb={1} sx={{ color: theme.palette.text.secondary }}>
              The following will be cleaned up before upgrading:
            </Typography>
            <Box component="ul" sx={{ margin: 0, paddingLeft: 0, listStyle: 'none', color: theme.palette.text.secondary, fontSize: '0.875rem' }}>
              {[
                'Delete MigrationPlans',
                'Delete RollingMigrationPlans',
                'Scale down Agents',
                'Delete VMware credentials',
                'Delete PCD credentials',
                'Delete Custom Resources'
              ].map((item) => (
                <Box component="li" key={item} sx={{ display: 'flex', alignItems: 'center', mb: 0.5 }}>
                  {cleanupCompleted && (
                    <CheckCircleIcon color="success" sx={{ mr: 1, fontSize: 18 }} />
                  )}
                  <Typography variant="body2" sx={{ color: theme.palette.text.secondary }}>{item}</Typography>
                </Box>
              ))}
            </Box>
            {cleanupCompleted && (
              <Box display="flex" alignItems="center" mt={1}>
                <CheckCircleIcon color="success" sx={{ mr: 1, fontSize: 18 }} />
                <Typography variant="body2" color="success.main">Cleanup completed</Typography>
              </Box>
            )}
          </Box>

          {/* Cleanup in progress */}
          {cleanUpInProgress && (
            <Box display="flex" flexDirection="column" alignItems="center" mb={2}>
              <CircularProgress size={32} />
              <Typography variant="body2" mt={2}>
                Cleaning up resources...
              </Typography>
            </Box>
          )}

          {/* Upgrade progress */}
          {upgradeInProgress && (
            <Box display="flex" flexDirection="column" alignItems="center" mb={2}>
              <CircularProgress size={32} />
              <Typography variant="body2" mt={2}>
                {getUIStatusMessage(progressData?.status)}
              </Typography>
            </Box>
          )}

          {(upgradeInProgress || cleanUpInProgress || upgradeMutation.isPending || cleanupMutation.isPending) && (
            <Alert severity="warning" sx={{ mt: 2 }}>
              Processing. Please do not close or refresh this page.
            </Alert>
          )}

          {errorMsg && (
            <Box display="flex" justifyContent="center" mb={2}>
              <Alert severity="error" sx={{ mb: 2, display: 'flex', alignItems: 'center' }}>
                {errorMsg}
              </Alert>
            </Box>
          )}

          {successMsg && (
            <Box display="flex" justifyContent="center" alignItems="center" mb={2}>
              <Alert
                severity="success"
                sx={{
                  mb: 2,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  textAlign: 'center',
                  width: '100%'
                }}
              >
                {successMsg}
              </Alert>
            </Box>
          )}

          {upgradeMutation.isPending && !upgradeInProgress && (
            <Box display="flex" justifyContent="center" mb={2}>
              <CircularProgress size={24} />
            </Box>
          )}
        </DialogContent>
        <DialogActions sx={{ gap: 1, p: 2 }}>
          <ActionButton
            onClick={() => {
              setSuccessMsg('')
              upgradeMutation.mutate()
            }}
            disabled={
              !selectedVersion ||
              upgradeInProgress ||
              cleanUpInProgress ||
              areVersionsLoading ||
              upgradeMutation.isPending
            }
            tone="primary"
            fullWidth
          >
            Upgrade
          </ActionButton>
          <Tooltip
            title={
              <Typography sx={{ fontSize: '0.875rem', whiteSpace: 'nowrap' }}>
                This will clean up all resources listed above
              </Typography>
            }
            arrow
          >
            <span style={{ width: '100%' }}>
              <ActionButton
                onClick={handleCleanup}
                tone="primary"
                fullWidth
                disabled={upgradeInProgress || cleanUpInProgress}
              >
                Cleanup
              </ActionButton>
            </span>
          </Tooltip>
          <ActionButton
            onClick={onClose}
            tone="secondary"
            fullWidth
            disabled={upgradeInProgress || cleanUpInProgress}
          >
            Cancel
          </ActionButton>
        </DialogActions>
      </Dialog>
    </React.Fragment>
  )
}