import { useEffect } from 'react'
import { RollingMigrationPlan } from 'src/api/rolling-migration-plans/model'
import { useAmplitude } from './useAmplitude'
import { useErrorHandler } from './useErrorHandler'
import { useStatusTracker } from './useStatusMonitor'
import { AMPLITUDE_EVENTS } from 'src/types/amplitude'

export const useRollingMigrationsStatusMonitor = (
  rollingMigrationPlans: RollingMigrationPlan[] = []
) => {
  const { track } = useAmplitude({ component: 'RollingMigrationsStatusMonitor' })
  const { reportError } = useErrorHandler({
    component: 'RollingMigrationsStatusMonitor'
  })
  const { statusTrackerRef, autoCleanup } = useStatusTracker<string>()

  useEffect(() => {
    if (!rollingMigrationPlans || rollingMigrationPlans.length === 0) return

    // Auto-cleanup old trackers
    autoCleanup(rollingMigrationPlans.map((plan) => plan.metadata?.name).filter(Boolean))

    rollingMigrationPlans.forEach((rollingMigrationPlan) => {
      const planName = rollingMigrationPlan.metadata?.name
      if (!planName) return

      const currentPhase = rollingMigrationPlan.status?.phase
      const tracker = statusTrackerRef.current[planName]

      // Initialize tracker for new rolling migration plans
      if (!tracker) {
        statusTrackerRef.current[planName] = {
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

      // Get error details from status
      const getErrorDetails = () => {
        return {
          message:
            rollingMigrationPlan.status?.migrationMessage ||
            `Rolling Migration Plan ${currentPhase}`,
          phase: currentPhase
        }
      }

      // Handle rolling migration plan execution failures
      const isFailed = currentPhase === 'Failed'
      if (isFailed && tracker.lastReportedPhase !== currentPhase) {
        const errorDetails = getErrorDetails()

        // Track with Amplitude
        track(AMPLITUDE_EVENTS.CLUSTER_CONVERSION_EXECUTION_FAILED, {
          rollingMigrationPlanName: planName,
          clusterName: rollingMigrationPlan.spec?.clusterSequence?.[0]?.clusterName,
          previousPhase: tracker.previousPhase,
          currentPhase,
          errorMessage: errorDetails.message,
          bmConfigRef: rollingMigrationPlan.spec?.bmConfigRef?.name,
          clusterSequenceLength: rollingMigrationPlan.spec?.clusterSequence?.length || 0,
          vmSequenceLength:
            rollingMigrationPlan.spec?.clusterSequence?.[0]?.vmSequence?.length || 0,
          namespace: rollingMigrationPlan.metadata?.namespace,
          migrationStrategy: rollingMigrationPlan.spec?.migrationStrategy?.type
        })

        // Report to Bugsnag
        const bugsnagError = new Error(
          `Rolling migration plan execution failed: ${errorDetails.message}`
        )
        reportError(bugsnagError, {
          context: 'rolling-migration-plan-execution-failure',
          metadata: {
            rollingMigrationPlanName: planName,
            clusterName: rollingMigrationPlan.spec?.clusterSequence?.[0]?.clusterName,
            previousPhase: tracker.previousPhase,
            currentPhase,
            errorMessage: errorDetails.message,
            bmConfigRef: rollingMigrationPlan.spec?.bmConfigRef?.name,
            clusterSequenceLength: rollingMigrationPlan.spec?.clusterSequence?.length || 0,
            vmSequenceLength:
              rollingMigrationPlan.spec?.clusterSequence?.[0]?.vmSequence?.length || 0,
            namespace: rollingMigrationPlan.metadata?.namespace,
            migrationStrategy: rollingMigrationPlan.spec?.migrationStrategy?.type,
            fullStatus: rollingMigrationPlan.status,
            action: 'rolling-migration-plan-execution-failed'
          }
        })

        console.error('Rolling migration plan execution failed:', {
          rollingMigrationPlanName: planName,
          errorDetails,
          rollingMigrationPlan
        })

        // Mark as reported BEFORE updating previousPhase to prevent race conditions
        statusTrackerRef.current[planName].lastReportedPhase = currentPhase
      }

      // Handle rolling migration plan success (optional - for analytics)
      const isSucceeded = currentPhase === 'Succeeded'
      if (isSucceeded && tracker.lastReportedPhase !== currentPhase) {
        track(AMPLITUDE_EVENTS.CLUSTER_CONVERSION_SUCCEEDED, {
          rollingMigrationPlanName: planName,
          clusterName: rollingMigrationPlan.spec?.clusterSequence?.[0]?.clusterName,
          previousPhase: tracker.previousPhase,
          currentPhase,
          bmConfigRef: rollingMigrationPlan.spec?.bmConfigRef?.name,
          clusterSequenceLength: rollingMigrationPlan.spec?.clusterSequence?.length || 0,
          vmSequenceLength:
            rollingMigrationPlan.spec?.clusterSequence?.[0]?.vmSequence?.length || 0,
          namespace: rollingMigrationPlan.metadata?.namespace,
          migrationStrategy: rollingMigrationPlan.spec?.migrationStrategy?.type
        })

        // Mark as reported BEFORE updating previousPhase to prevent race conditions
        statusTrackerRef.current[planName].lastReportedPhase = currentPhase
      }

      // Update previous phase (only after all event tracking is complete)
      if (statusTrackerRef.current[planName]) {
        statusTrackerRef.current[planName].previousPhase = currentPhase
      }
    })
  }, [rollingMigrationPlans, track, reportError, autoCleanup])

  return {}
}
