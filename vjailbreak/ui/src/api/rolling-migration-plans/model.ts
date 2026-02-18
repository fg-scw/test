export interface GetRollingMigrationPlansList {
  apiVersion: string
  items: RollingMigrationPlan[]
  kind: string
  metadata: ListMetadata
}

export interface RollingMigrationPlan {
  apiVersion: string
  kind: string
  metadata: ItemMetadata
  spec: RollingMigrationPlanSpec
  status?: RollingMigrationPlanStatus
}

export interface ListMetadata {
  continue: string
  resourceVersion: string
}

export interface ItemMetadata {
  annotations?: Record<string, string>
  creationTimestamp?: Date
  generation?: number
  name: string
  namespace?: string
  resourceVersion?: string
  uid?: string
  labels?: Record<string, string>
}

export interface RollingMigrationPlanSpec {
  clusterSequence: ClusterSequence[]
  clusterMapping: ClusterMapping[]
  bmConfigRef: BMConfigRef
  advancedOptions?: Record<string, unknown>
  firstBootScript?: string
  migrationStrategy?: MigrationStrategy
  migrationTemplate?: string
}

export interface ClusterSequence {
  clusterName: string
  vmSequence: VMSequence[]
}

export interface ClusterMapping {
  vmwareClusterName: string
  pcdClusterName: string
}

export interface VMSequence {
  vmName: string
  esxiName: string
}

export interface BMConfigRef {
  name: string
}

export interface MigrationStrategy {
  type: string
  adminInitiatedCutOver?: boolean
  healthCheckPort?: string
  performHealthChecks?: boolean
  dataCopyStart?: string
  vmCutoverStart?: string
  vmCutoverEnd?: string
}

export interface RollingMigrationPlanStatus {
  migrationStatus?: string
  migrationMessage?: string
  phase?: string
  startTime?: string
  completionTime?: string
}
