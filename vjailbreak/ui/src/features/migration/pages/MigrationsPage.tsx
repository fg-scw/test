import { useState, useEffect } from 'react'
import Snackbar from '@mui/material/Snackbar'
import Alert from '@mui/material/Alert'
import { useQueryClient } from '@tanstack/react-query'
import { FIVE_SECONDS, THIRTY_SECONDS } from 'src/constants'
import { useMigrationsQuery, MIGRATIONS_QUERY_KEY } from '../hooks/useMigrationsQuery'
import { deleteMigration } from '../api/migrations'
import { getMigrationPlan, patchMigrationPlan } from '../api/migrationPlans'
import { ConfirmationDialog } from 'src/components/dialogs'
import { Migration } from '../api/migrations'
import MigrationsTable from '../components/MigrationsTable'
import WarningIcon from '@mui/icons-material/Warning'
import { useMigrationStatusMonitor } from '../hooks/useMigrationStatusMonitor'

export default function MigrationsPage() {
  const queryClient = useQueryClient()
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [selectedMigrations, setSelectedMigrations] = useState<Migration[]>([])
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const [openSnackbar, setOpenSnackbar] = useState(false)

  const { data: migrations, refetch: refetchMigrations } = useMigrationsQuery(undefined, {
    refetchInterval: (query) => {
      const migrations = query?.state?.data || []
      const hasPendingMigration = !!migrations.find((m) => m.status === undefined)
      return hasPendingMigration ? FIVE_SECONDS : THIRTY_SECONDS
    },
    staleTime: 0,
    refetchOnMount: true
  })

  useMigrationStatusMonitor(migrations)
  useEffect(() => {
    const showSuccess = sessionStorage.getItem('showUpgradeSuccess')
    if (showSuccess === 'true') {
      setOpenSnackbar(true)
      sessionStorage.removeItem('showUpgradeSuccess')
    }
  }, [])

  const handleDeleteClick = (migrationName: string) => {
    const migration = migrations?.find((m) => m.metadata.name === migrationName)
    if (migration) {
      setSelectedMigrations([migration])
      setDeleteDialogOpen(true)
    }
  }

  const handleDeleteClose = () => {
    setDeleteDialogOpen(false)
    setSelectedMigrations([])
    setDeleteError(null)
  }

  const handleDeleteSelected = (migrations: Migration[]) => {
    setSelectedMigrations(migrations)
    setDeleteDialogOpen(true)
  }

  const handleDeleteMigration = async (migrations: Migration[]) => {
    const migrationPlanUpdates = migrations.reduce(
      (acc, migration) => {
        const planId = migration.spec.migrationPlan
        if (!acc[planId]) {
          acc[planId] = {
            vmsToRemove: new Set<string>(),
            migrationsToDelete: new Set<string>()
          }
        }
        acc[planId].vmsToRemove.add(migration.spec.vmName)
        acc[planId].migrationsToDelete.add(migration.metadata.name)
        return acc
      },
      {} as Record<string, { vmsToRemove: Set<string>; migrationsToDelete: Set<string> }>
    )

    await Promise.all(
      Object.entries(migrationPlanUpdates).map(
        async ([planId, { vmsToRemove, migrationsToDelete }]) => {
          const migrationPlan = await getMigrationPlan(planId)
          const updatedVirtualMachines = migrationPlan.spec.virtualMachines?.[0]?.filter(
            (vm) => !vmsToRemove.has(vm)
          )

          await patchMigrationPlan(planId, {
            spec: {
              virtualMachines: [updatedVirtualMachines]
            }
          })

          await Promise.all(
            Array.from(migrationsToDelete).map((migrationName) => deleteMigration(migrationName))
          )
        }
      )
    )

    queryClient.invalidateQueries({ queryKey: MIGRATIONS_QUERY_KEY })
    handleDeleteClose()
  }

  const handleCloseSnackbar = (_event?: React.SyntheticEvent | Event, reason?: string) => {
    if (reason === 'clickaway') {
      return
    }
    setOpenSnackbar(false)
  }

  const getCustomErrorMessage = (error: Error | string) => {
    const baseMessage = 'Failed to delete migrations'
    if (error instanceof Error) {
      return `${baseMessage}: ${error.message}`
    }
    return baseMessage
  }

  return (
    <>
      <MigrationsTable
        refetchMigrations={refetchMigrations}
        migrations={migrations || []}
        onDeleteMigration={handleDeleteClick}
        onDeleteSelected={handleDeleteSelected}
      />

      <ConfirmationDialog
        open={deleteDialogOpen}
        onClose={handleDeleteClose}
        title="Confirm Delete Migration"
        icon={<WarningIcon color="warning" />}
        message={
          selectedMigrations.length > 1
            ? 'Are you sure you want to delete these migrations?'
            : `Are you sure you want to delete migration "${selectedMigrations[0]?.metadata.name}"?`
        }
        items={selectedMigrations.map((m) => ({
          id: m.metadata.name,
          name: m.metadata.name
        }))}
        actionLabel="Delete"
        actionColor="error"
        actionVariant="outlined"
        onConfirm={() => handleDeleteMigration(selectedMigrations)}
        customErrorMessage={getCustomErrorMessage}
        errorMessage={deleteError}
        onErrorChange={setDeleteError}
      />
      <Snackbar
        open={openSnackbar}
        autoHideDuration={6000}
        onClose={handleCloseSnackbar}
        anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
      >
        <Alert onClose={handleCloseSnackbar} severity="success" sx={{ width: '100%' }}>
          Successfully upgraded to {sessionStorage.getItem('upgradedVersion')}!
        </Alert>
      </Snackbar>
    </>
  )
}
