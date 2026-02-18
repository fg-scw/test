export interface GetClusterMigrationsList {
  apiVersion: string
  items: ClusterMigration[]
  kind: string
  metadata: GetClusterMigrationsListMetadata
}

export interface ClusterMigration {
  apiVersion: string
  kind: string
  metadata: ClusterMigrationMetadata
  spec: ClusterMigrationSpec
  status: ClusterMigrationStatus
}

export interface ClusterMigrationMetadata {
  creationTimestamp: string
  finalizers: string[]
  generation: number
  name: string
  namespace: string
  ownerReferences: OwnerReference[]
  resourceVersion: string
  uid: string
}

export interface OwnerReference {
  apiVersion: string
  kind: string
  name: string
  uid: string
}

export interface ClusterMigrationSpec {
  clusterName: string
  esxiMigrationSequence: string[]
  openstackCredsRef: NameReference
  rollingMigrationPlanRef: NameReference
  vmwareCredsRef: NameReference
}

export interface NameReference {
  name: string
}

export interface ClusterMigrationStatus {
  currentESXi: string
  message: string
  phase: string
}

export interface GetClusterMigrationsListMetadata {
  continue: string
  resourceVersion: string
}
