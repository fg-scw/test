import { GridColDef, GridRowSelectionModel, GridToolbarContainer } from '@mui/x-data-grid'
import { Button, Typography, Box, IconButton, Tooltip } from '@mui/material'
import DeleteIcon from '@mui/icons-material/DeleteOutlined'
import MigrationIcon from '@mui/icons-material/SwapHoriz'
import ReplayIcon from '@mui/icons-material/Replay'
import FiberManualRecordIcon from '@mui/icons-material/FiberManualRecord'
import { useCallback, useMemo, useState } from 'react'
import { CustomSearchToolbar } from 'src/components/grid'
import { CommonDataGrid } from 'src/components/grid'
import ListAltIcon from '@mui/icons-material/ListAlt'
import { LogsDrawer } from '.'
import { Condition, Migration, Phase } from '../api/migrations'
import MigrationDetailModal from 'src/components/migrations/MigrationDetailModal'
import MigrationProgress from '../components/MigrationProgress'
import { QueryObserverResult } from '@tanstack/react-query'
import { RefetchOptions } from '@tanstack/react-query'
import { calculateTimeElapsed, formatDateTime } from 'src/utils'
import { TriggerAdminCutoverButton } from '.'
import PlayArrowIcon from '@mui/icons-material/PlayArrow'
import { triggerAdminCutover, deleteMigration } from '../api/migrations'
import { ConfirmationDialog } from 'src/components/dialogs'
import { keyframes } from '@mui/material/styles'

const pulse = keyframes`
  0% {
    opacity: 1;
  }
  50% {
    opacity: 0.4;
  }
  100% {
    opacity: 1;
  }
`
import { TooltipContent, ClickableTableCell } from 'src/components'
import { useMigrationPlanDestinationsQuery } from '../api/useMigrationPlanDestinationsQuery'

const STATUS_ORDER = {
  Running: 0,
  Failed: 1,
  Succeeded: 2,
  Pending: 3
}
const PHASE_STEPS = {
  [Phase.Pending]: 1,
  [Phase.Validating]: 2,
  [Phase.AwaitingDataCopyStart]: 3,
  [Phase.CopyingBlocks]: 4,
  [Phase.CopyingChangedBlocks]: 5,
  [Phase.ConvertingDisk]: 6,
  [Phase.AwaitingCutOverStartTime]: 7,
  [Phase.AwaitingAdminCutOver]: 8,
  [Phase.Succeeded]: 9,
  [Phase.Failed]: 10,
  [Phase.ValidationFailed]: 11
}

const IN_PROGRESS_PHASES = [
  Phase.Pending,
  Phase.Validating,
  Phase.AwaitingDataCopyStart,
  Phase.CopyingBlocks,
  Phase.CopyingChangedBlocks,
  Phase.ConvertingDisk,
  Phase.AwaitingCutOverStartTime,
  Phase.AwaitingAdminCutOver
]

const getProgressText = (
  phase: Phase | undefined,
  conditions: Condition[] | undefined,
  currentDisk?: string,
  totalDisks?: number
) => {
  if (!phase || phase === Phase.Unknown) {
    return 'Unknown Status'
  }

  const stepNumber = PHASE_STEPS[phase] || 0
  const totalSteps = 9

  const latestCondition = conditions?.sort(
    (a, b) => new Date(b.lastTransitionTime).getTime() - new Date(a.lastTransitionTime).getTime()
  )[0]

  const message = latestCondition?.message || phase

  if (phase === Phase.Failed || phase === Phase.ValidationFailed || phase === Phase.Succeeded) {
    return `${phase} - ${message}`
  }

  let diskInfo = ''
  if (currentDisk && totalDisks && (phase === Phase.CopyingBlocks || phase === Phase.CopyingChangedBlocks)) {
    const parsedDisk = parseInt(currentDisk, 10)
    const currentDiskNum = Number.isNaN(parsedDisk) ? 1 : parsedDisk + 1
    diskInfo = ` (disk ${currentDiskNum}/${totalDisks})`
  }

  let progressText = `STEP ${stepNumber}/${totalSteps}: ${phase}${diskInfo} - ${message}`

  return progressText
}


interface CustomToolbarProps {
  numSelected: number
  onDeleteSelected: () => void
  onBulkAdminCutover: () => void
  numEligibleForCutover: number
  refetchMigrations: (options?: RefetchOptions) => Promise<QueryObserverResult<Migration[], Error>>
  onStatusFilterChange: (filter: string) => void
  currentStatusFilter: string
  onDateFilterChange: (filter: string) => void
  currentDateFilter: string
}

const CustomToolbar = ({
  numSelected,
  onDeleteSelected,
  onBulkAdminCutover,
  numEligibleForCutover,
  refetchMigrations,
  onStatusFilterChange,
  currentStatusFilter,
  onDateFilterChange,
  currentDateFilter
}: CustomToolbarProps) => {
  return (
    <GridToolbarContainer
      sx={{
        p: 2,
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center'
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <MigrationIcon />
        <Typography variant="h6" component="h2">
          Migrations
        </Typography>
      </Box>
      <Box sx={{ display: 'flex', gap: 2, alignItems: 'center' }}>
        {numSelected > 0 ? (
          <>
            <Button
              variant="outlined"
              color="error"
              startIcon={<DeleteIcon />}
              onClick={onDeleteSelected}
              sx={{ height: 40 }}
            >
              Delete Selected ({numSelected})
            </Button>

            {numEligibleForCutover > 0 && (
              <Button
                variant="outlined"
                color="primary"
                startIcon={<PlayArrowIcon />}
                onClick={onBulkAdminCutover}
                sx={{ height: 40 }}
              >
                Trigger Cutover ({numEligibleForCutover})
              </Button>
            )}
          </>
        ) : null}
        <CustomSearchToolbar
          placeholder="Search by Name, Status, or Progress"
          onRefresh={refetchMigrations}
          onStatusFilterChange={numSelected === 0 ? onStatusFilterChange : undefined}
          currentStatusFilter={currentStatusFilter}
          onDateFilterChange={numSelected === 0 ? onDateFilterChange : undefined}
          currentDateFilter={currentDateFilter}
        />
      </Box>
    </GridToolbarContainer>
  )
}

interface MigrationsTableProps {
  migrations: Migration[]
  onDeleteMigration?: (name: string) => void
  onDeleteSelected?: (migrations: Migration[]) => void
  refetchMigrations: (options?: RefetchOptions) => Promise<QueryObserverResult<Migration[], Error>>
}

export default function MigrationsTable({
  migrations,
  onDeleteMigration,
  onDeleteSelected,
  refetchMigrations
}: MigrationsTableProps) {
  const [selectedRows, setSelectedRows] = useState<GridRowSelectionModel>([])
  const [isBulkCutoverLoading, setIsBulkCutoverLoading] = useState(false)
  const [bulkCutoverDialogOpen, setBulkCutoverDialogOpen] = useState(false)
  const [bulkCutoverError, setBulkCutoverError] = useState<string | null>(null)
  const [migrationDetailModalOpen, setMigrationDetailModalOpen] = useState(false)
  const [selectedMigrationDetail, setSelectedMigrationDetail] = useState<Migration | null>(null)
  const [statusFilter, setStatusFilter] = useState('All')
  const [dateFilter, setDateFilter] = useState('All Time')
  const [logsDrawerOpen, setLogsDrawerOpen] = useState(false)
  const [selectedPod, setSelectedPod] = useState<{
    name: string
    namespace: string
    migrationName?: string
  } | null>(null)

  const handleSelectionChange = useCallback((newSelection: GridRowSelectionModel) => {
    setSelectedRows(newSelection)
  }, [])

  const filteredMigrations = useMemo(() => {
    if (!migrations) return []

    const now = new Date()
    let timeCutoff = 0

    switch (dateFilter) {
      case 'Last 24 hours':
        timeCutoff = now.getTime() - 24 * 60 * 60 * 1000
        break
      case 'Last 7 days':
        timeCutoff = now.getTime() - 7 * 24 * 60 * 60 * 1000
        break
      case 'Last 30 days':
        timeCutoff = now.getTime() - 30 * 24 * 60 * 60 * 1000
        break
      default:
        timeCutoff = 0
    }

    const dateFiltered = migrations.filter((m) => {
      if (!m.metadata?.creationTimestamp) return false
      return new Date(m.metadata.creationTimestamp).getTime() >= timeCutoff
    })

    switch (statusFilter) {
      case 'Succeeded':
        return dateFiltered.filter((m) => m.status?.phase === Phase.Succeeded)
      case 'Failed':
        return dateFiltered.filter((m) => m.status?.phase === Phase.Failed)
      case 'In Progress':
        return dateFiltered.filter(
          (m) => m.status?.phase && IN_PROGRESS_PHASES.includes(m.status.phase)
        )
      case 'All':
      default:
        return dateFiltered
    }
  }, [migrations, statusFilter, dateFilter])

  const destinationByPlanQuery = useMigrationPlanDestinationsQuery(filteredMigrations)

  const destinationByPlan = destinationByPlanQuery.data || {}

  const columns: GridColDef[] = useMemo(() => {
    return [
      {
        field: 'name',
        headerName: 'Name',
        flex: 0.7,
        valueGetter: (_, row) => row.spec?.vmName,
        renderCell: (params) => {
          const vmName = params.row?.spec?.vmName || '-'
          const migrationType = params.row?.spec?.migrationType
          const phase = params.row?.status?.phase
          const isHotMigration = migrationType?.toLowerCase() === 'hot'
          const isColdMigration = migrationType?.toLowerCase() === 'cold'
          const isMockMigration = migrationType?.toLowerCase() === 'mock'
          
          const namespace = params.row.metadata?.namespace
          const planName = params.row.spec?.migrationPlan || params.row.metadata?.labels?.migrationplan
          const key = namespace && planName ? `${namespace}::${planName}` : ''
          const destination = key ? destinationByPlan[key] : null

          const destinationTenant = destination?.destinationTenant || 'N/A'
          const destinationCluster = destination?.destinationCluster || 'N/A'

          const tooltipTitle = (
            <TooltipContent title="Destination" lines={[`Tenant: ${destinationTenant}`, `Cluster: ${destinationCluster}`]} />
          )

          // Logic for the blinking pulse
          const activePhases = new Set([
            Phase.Pending, Phase.Validating, Phase.AwaitingDataCopyStart,
            Phase.CopyingBlocks, Phase.CopyingChangedBlocks, Phase.ConvertingDisk,
            Phase.AwaitingCutOverStartTime, Phase.AwaitingAdminCutOver, Phase.Unknown
          ])
          const isInProgress = activePhases.has(phase)
          const syncedPulse = `${pulse} 2s ease-in-out -20s infinite` 

          return (
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
              {isHotMigration && (
                <Tooltip title="Hot Migration">
                  <FiberManualRecordIcon 
                    sx={{ 
                      fontSize: 12, color: '#FFAE42',
                      ...(isInProgress && { animation: syncedPulse })
                    }} 
                  />
                </Tooltip>
              )}
              {isColdMigration && (
                <Tooltip title="Cold Migration">
                  <FiberManualRecordIcon 
                    sx={{ 
                      fontSize: 12, color: '#4293FF',
                      ...(isInProgress && { animation: syncedPulse })
                    }} 
                  />
                </Tooltip>
              )}
              {isMockMigration && (
                <Tooltip title="Migration without poweroff">
                  <FiberManualRecordIcon 
                    sx={{ 
                      fontSize: 12, color: '#9e1111ff',
                      ...(isInProgress && { animation: syncedPulse })
                    }} 
                  />
                </Tooltip>
              )}
              <ClickableTableCell
                tooltipTitle={tooltipTitle}
                onClick={() => {
                  params.row.setSelectedMigrationDetail?.(params.row)
                  params.row.setMigrationDetailModalOpen?.(true)
                }}
              >
                {vmName}
              </ClickableTableCell>
            </Box>
          )
        }
      },
      {
        field: 'status',
        headerName: 'Status',
        valueGetter: (_, row) => row?.status?.phase || 'Pending',
        flex: 0.5,
        sortComparator: (v1, v2) => {
          const order1 = STATUS_ORDER[v1] ?? Number.MAX_SAFE_INTEGER
          const order2 = STATUS_ORDER[v2] ?? Number.MAX_SAFE_INTEGER
          return order1 - order2
        }
      },
      {
        field: 'agent',
        headerName: 'Agent',
        valueGetter: (_, row) => row.status?.agentName,
        flex: 1
      },
      {
        field: 'timeElapsed',
        headerName: 'Time Elapsed',
        valueGetter: (_, row) => calculateTimeElapsed(row.metadata?.creationTimestamp, row.status),
        flex: 0.8,
        renderCell: (params) => {
          const createdAt = formatDateTime(params.row.metadata?.creationTimestamp)
          const tooltip = createdAt === '-' ? 'Created at: N/A' : `Created at: ${createdAt}` 
          return (
            <Tooltip title={tooltip} arrow>
              <Typography variant="body2" sx={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                {String(params.value ?? '-')}
              </Typography>
            </Tooltip>
          )
        }
      },
      {
        field: 'createdAt',
        headerName: 'Created At',
        valueGetter: (_, row) => formatDateTime(row.metadata?.creationTimestamp),
        flex: 1
      },
      {
        field: 'status.conditions',
        headerName: 'Progress',
        valueGetter: (_, row) =>
          getProgressText(row.status?.phase, row.status?.conditions, row.status?.currentDisk, row.status?.totalDisks),
        flex: 2,
        renderCell: (params) => {
          const phase = params.row?.status?.phase
          const conditions = params.row?.status?.conditions
          const currentDisk = params.row?.status?.currentDisk
          const totalDisks = params.row?.status?.totalDisks
          return conditions ? (
            <MigrationProgress
              phase={phase}
              progressText={getProgressText(phase, conditions, currentDisk, totalDisks)}
            />
          ) : null
        }
      },
      {
        field: 'actions',
        headerName: 'Actions',
        flex: 1,
        renderCell: (params) => {
          const phase = params.row?.status?.phase
          const initiateCutover = params.row?.spec?.initiateCutover
          const migrationName = params.row?.metadata?.name
          const namespace = params.row?.metadata?.namespace
          const retryable = params.row?.status?.retryable
          const showRetryButton = phase === Phase.Failed
          const isRetryDisabled = retryable === false

          const handleRetry = async () => {
            if (!migrationName || !namespace) return
            try {
              await deleteMigration(migrationName, namespace)
              params.row.refetchMigrations?.()
            } catch (error) {
              console.error('Failed retry:', error)
            }
          }

          const showAdminCutover = initiateCutover && phase === Phase.AwaitingAdminCutOver

          return (
            <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
              {params.row.spec?.podRef && (
                <Tooltip title="View pod logs">
                  <IconButton
                    onClick={(e) => {
                      e.stopPropagation()
                      params.row.setSelectedPod({
                        name: params.row.spec.podRef,
                        namespace: params.row.metadata?.namespace || '',
                        migrationName: params.row.metadata?.name || ''
                      })
                      params.row.setLogsDrawerOpen(true)
                    }}
                    size="small"
                  >
                    <ListAltIcon />
                  </IconButton>
                </Tooltip>
              )}
              {showAdminCutover && (
                <TriggerAdminCutoverButton
                  migrationName={migrationName}
                  onSuccess={() => params.row.refetchMigrations?.()}
                />
              )}
              {showRetryButton && (
                <Tooltip
                  title={
                    isRetryDisabled
                      ? "This migration cannot be retried because the VM has RDM disks. To retry, manually restart the migration."
                      : "Retry migration"
                  }
                >
                  <span>
                    <IconButton
                      onClick={(e) => {
                        e.stopPropagation()
                        if (!isRetryDisabled) {
                          handleRetry()
                        }
                      }}
                      size="small"
                      disabled={isRetryDisabled}
                      sx={{
                        cursor: isRetryDisabled ? 'not-allowed' : 'pointer',
                        position: 'relative',
                        '&.Mui-disabled': {
                          opacity: 0.4
                        }
                      }}
                    >
                      <ReplayIcon />
                    </IconButton>
                  </span>
                </Tooltip>
              )}
              <Tooltip title={'Delete migration'}>
                <IconButton
                  onClick={(e) => {
                    e.stopPropagation()
                    params.row.onDelete(params.row.metadata?.name)
                  }}
                  size="small"
                >
                  <DeleteIcon />
                </IconButton>
              </Tooltip>
            </Box>
          )
        }
      }
    ]
  }, [destinationByPlan, pulse])

  const selectedMigrations = useMemo(
    () => migrations?.filter((m) => selectedRows.includes(m.metadata?.name)) || [],
    [migrations, selectedRows]
  )
  const eligibleForCutover = useMemo(
    () => selectedMigrations.filter((migration) => migration.status?.phase === Phase.AwaitingAdminCutOver),
    [selectedMigrations]
  )

  const handleBulkAdminCutover = useCallback(async () => {
    if (eligibleForCutover.length === 0) return

    setBulkCutoverError(null)
    setIsBulkCutoverLoading(true)

    try {
      await Promise.all(
        eligibleForCutover.map(async (migration) => {
          const result = await triggerAdminCutover(
            'migration-system',
            migration.metadata?.name || ''
          )
          if (!result.success) {
            throw new Error(result.message)
          }
          return result
        })
      )

      await refetchMigrations()

      setSelectedRows([])
    } catch (error) {
      console.error('Failed to trigger bulk admin cutover:', error)
      const errorMessage =
        error instanceof Error ? error.message : 'Failed to trigger bulk admin cutover'
      setBulkCutoverError(errorMessage)
      throw error
    } finally {
      setIsBulkCutoverLoading(false)
    }
  }, [eligibleForCutover, refetchMigrations])

  const handleCloseBulkCutoverDialog = useCallback(() => {
    if (!isBulkCutoverLoading) {
      setBulkCutoverDialogOpen(false)
      setBulkCutoverError(null)
    }
  }, [isBulkCutoverLoading])

  const migrationsWithActions = useMemo(
    () =>
      filteredMigrations?.map((migration) => ({
        ...migration,
        onDelete: onDeleteMigration,
        refetchMigrations,
        setSelectedPod,
        setLogsDrawerOpen,
        setMigrationDetailModalOpen,
        setSelectedMigrationDetail
      })) || [],
    [
      filteredMigrations,
      onDeleteMigration,
      refetchMigrations,
      setLogsDrawerOpen,
      setMigrationDetailModalOpen,
      setSelectedMigrationDetail,
      setSelectedPod
    ]
  )

  return (
    <>
      <CommonDataGrid
        rows={migrationsWithActions}
        columns={
          onDeleteSelected === undefined && onDeleteMigration === undefined
            ? columns.filter((column) => column.field !== 'actions')
            : columns
        }
        initialState={{
          pagination: { paginationModel: { page: 0, pageSize: 25 } },
          sorting: {
            sortModel: [{ field: 'status', sort: 'asc' }]
          },
          columns: {
            columnVisibilityModel: {
              createdAt: false
            }
          }
        }}
        pageSizeOptions={[25, 50, 100]}
        localeText={{ noRowsLabel: 'No Migrations Available' }}
        emptyMessage="No migrations available"
        getRowId={(row) => row.metadata?.name}
        checkboxSelection={onDeleteSelected !== undefined && onDeleteMigration !== undefined}
        onRowSelectionModelChange={handleSelectionChange}
        rowSelectionModel={selectedRows}
        disableRowSelectionOnClick
        loading={isBulkCutoverLoading}
        slots={{
          toolbar:
            onDeleteSelected !== undefined && onDeleteMigration !== undefined
              ? () => (
                  <CustomToolbar
                    numSelected={selectedRows.length}
                    numEligibleForCutover={eligibleForCutover.length}
                    onDeleteSelected={() => {
                      const selectedMigrations = migrations?.filter((m) =>
                        selectedRows.includes(m.metadata?.name)
                      )
                      if (onDeleteSelected) {
                        onDeleteSelected(selectedMigrations || [])
                      }
                    }}
                    onBulkAdminCutover={() => setBulkCutoverDialogOpen(true)}
                    refetchMigrations={refetchMigrations}
                    onStatusFilterChange={setStatusFilter}
                    currentStatusFilter={statusFilter}
                    onDateFilterChange={setDateFilter}
                    currentDateFilter={dateFilter}
                  />
                )
              : undefined
        }}
      />

      <ConfirmationDialog
        open={bulkCutoverDialogOpen}
        onClose={handleCloseBulkCutoverDialog}
        title="Confirm Admin Cutover"
        icon={<PlayArrowIcon color="primary" />}
        message={
          eligibleForCutover.length > 1
            ? `Are you sure you want to trigger admin cutover for these ${eligibleForCutover.length} migrations?\n\n${eligibleForCutover
                .map((m) => `• ${m.metadata?.name}`)
                .join('\n')}\n\nThis will start the cutover process and cannot be undone.`
            : `Are you sure you want to trigger admin cutover for migration "${
                eligibleForCutover[0]?.metadata?.name
              }"?\n\nThis will start the cutover process and cannot be undone.`
        }
        items={eligibleForCutover.map((migration) => ({
          id: migration.metadata?.name || '',
          name: migration.metadata?.name || ''
        }))}
        actionLabel="Trigger Cutover"
        actionColor="primary"
        actionVariant="contained"
        onConfirm={handleBulkAdminCutover}
        errorMessage={bulkCutoverError}
        onErrorChange={setBulkCutoverError}
      />

      <LogsDrawer
        open={logsDrawerOpen}
        onClose={() => setLogsDrawerOpen(false)}
        podName={selectedPod?.name || ''}
        namespace={selectedPod?.namespace || ''}
        migrationName={selectedPod?.migrationName || ''}
      />

      <MigrationDetailModal
        open={migrationDetailModalOpen}
        migration={selectedMigrationDetail}
        onClose={() => setMigrationDetailModalOpen(false)}
      />
    </>
  )
}
