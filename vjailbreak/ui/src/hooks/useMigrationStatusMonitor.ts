import { useEffect } from 'react'
import { Migration, Phase } from 'src/api/migrations/model'
import { useAmplitude } from './useAmplitude'
import { useErrorHandler } from './useErrorHandler'
import { useStatusTracker } from './useStatusMonitor'
import { AMPLITUDE_EVENTS } from 'src/types/amplitude'

export const useMigrationStatusMonitor = (migrations: Migration[] = []) => {
  const { track } = useAmplitude({ component: 'MigrationStatusMonitor' })
  const { reportError } = useErrorHandler({
    component: 'MigrationStatusMonitor'
  })
  const { statusTrackerRef, autoCleanup } = useStatusTracker<Phase>()

  useEffect(() => {
    if (!migrations || migrations.length === 0) return

    // Auto-cleanup old trackers
    autoCleanup(migrations.map((m) => m.metadata?.name))

    migrations.forEach((migration) => {
      const migrationName = migration.metadata?.name
      if (!migrationName) return

      const currentPhase = migration.status?.phase
      const tracker = statusTrackerRef.current[migrationName]

      // Initialize tracker for new migrations
      if (!tracker) {
        statusTrackerRef.current[migrationName] = {
          previousPhase: currentPhase
        }
        return
      }

      // Skip if phase hasn't changed
      if (tracker.previousPhase === currentPhase) {
        return
      }

      // Skip if this phase has already been reported (prevents duplicate events)
      if (tracker.lastReportedPhase === currentPhase) {
        return
      }

      // Get error details from conditions
      const getErrorDetails = () => {
        const conditions = migration.status?.conditions || []
        // Find latest condition without sorting entire array
        const latestCondition =
          conditions.length > 0
            ? conditions.reduce((latest, current) => {
                const currentTime = new Date(current.lastTransitionTime).getTime()
                const latestTime = new Date(latest.lastTransitionTime).getTime()
                return currentTime > latestTime ? current : latest
              })
            : null

        return {
          message: latestCondition?.message || `Migration ${currentPhase}`,
          reason: latestCondition?.reason || 'Unknown',
          lastTransitionTime: latestCondition?.lastTransitionTime
        }
      }

      // Handle migration execution failures
      if (currentPhase === Phase.Failed && tracker.lastReportedPhase !== Phase.Failed) {
        const errorDetails = getErrorDetails()

        // Track with Amplitude
        track(AMPLITUDE_EVENTS.MIGRATION_EXECUTION_FAILED, {
          migrationName,
          migrationPlan: migration.spec?.migrationPlan,
          vmName: migration.spec?.vmName,
          podRef: migration.spec?.podRef,
          previousPhase: tracker.previousPhase,
          currentPhase,
          errorMessage: errorDetails.message,
          errorReason: errorDetails.reason,
          failureTime: errorDetails.lastTransitionTime,
          namespace: migration.metadata?.namespace
        })

        // Report to Bugsnag
        const bugsnagError = new Error(`Migration execution failed: ${errorDetails.message}`)
        reportError(bugsnagError, {
          context: 'migration-execution-failure',
          metadata: {
            migrationName,
            migrationPlan: migration.spec?.migrationPlan,
            vmName: migration.spec?.vmName,
            podRef: migration.spec?.podRef,
            previousPhase: tracker.previousPhase,
            currentPhase,
            errorMessage: errorDetails.message,
            errorReason: errorDetails.reason,
            failureTime: errorDetails.lastTransitionTime,
            namespace: migration.metadata?.namespace,
            conditions: migration.status?.conditions,
            action: 'migration-execution-failed'
          }
        })

        console.error('Migration execution failed:', {
          migrationName,
          errorDetails,
          migration
        })

        // Mark as reported
        statusTrackerRef.current[migrationName].lastReportedPhase = Phase.Failed
      }

      // Handle migration success (optional - for analytics)
      if (currentPhase === Phase.Succeeded && tracker.lastReportedPhase !== Phase.Succeeded) {
        track(AMPLITUDE_EVENTS.MIGRATION_SUCCEEDED, {
          migrationName,
          migrationPlan: migration.spec?.migrationPlan,
          vmName: migration.spec?.vmName,
          previousPhase: tracker.previousPhase,
          currentPhase,
          namespace: migration.metadata?.namespace
        })

        // Mark as reported BEFORE updating previousPhase to prevent race conditions
        statusTrackerRef.current[migrationName].lastReportedPhase = Phase.Succeeded
      }

      // Update previous phase (only after all event tracking is complete)
      statusTrackerRef.current[migrationName].previousPhase = currentPhase
    })
  }, [migrations, track, reportError, autoCleanup])

  return {}
}
