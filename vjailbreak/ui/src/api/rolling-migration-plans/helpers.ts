import { v4 as uuidv4 } from 'uuid'
import {
  VMSequence,
  MigrationStrategy,
  BMConfigRef,
  ClusterMapping,
  RollingMigrationPlanSpec
} from './model'

interface CreateRollingMigrationPlanParams {
  name?: string
  clusterName: string
  vms: VMSequence[]
  clusterMapping: ClusterMapping[]
  bmConfigRef: BMConfigRef
  advancedOptions?: Record<string, unknown>
  firstBootScript?: string
  migrationStrategy?: MigrationStrategy
  migrationTemplate?: string
  namespace?: string
}

export const createRollingMigrationPlanJson = (params: CreateRollingMigrationPlanParams) => {
  const {
    name,
    clusterName,
    vms,
    clusterMapping,
    bmConfigRef,
    advancedOptions,
    firstBootScript,
    migrationStrategy,
    migrationTemplate,
    namespace
  } = params || {}

  const spec: Partial<RollingMigrationPlanSpec> = {
    clusterSequence: [
      {
        clusterName,
        vmSequence: vms
      }
    ],
    clusterMapping,
    bmConfigRef
  }

  // Add optional fields if they exist
  if (advancedOptions) spec.advancedOptions = advancedOptions
  if (firstBootScript) spec.firstBootScript = firstBootScript
  if (migrationStrategy) spec.migrationStrategy = migrationStrategy
  if (migrationTemplate) spec.migrationTemplate = migrationTemplate

  return {
    apiVersion: 'vjailbreak.k8s.pf9.io/v1alpha1',
    kind: 'RollingMigrationPlan',
    metadata: {
      name: name || `rolling-migration-plan-${uuidv4().substring(0, 8)}`,
      namespace: namespace,
      labels: {
        'app.kubernetes.io/name': 'migration',
        'app.kubernetes.io/part-of': 'vjailbreak'
      }
    },
    spec
  }
}
