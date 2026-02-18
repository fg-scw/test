export interface GetESXIMigrationsList {
  apiVersion: string
  items: ESXIMigration[]
  kind: string
  metadata: GetESXIMigrationsListMetadata
}

export interface ESXIMigration {
  apiVersion: string
  kind: string
  metadata: ESXIMigrationMetadata
  spec: ESXIMigrationSpec
  status: ESXIMigrationStatus
}

export interface ESXIMigrationMetadata {
  creationTimestamp: string
  finalizers: string[]
  generation: number
  name: string
  namespace: string
  ownerReferences: OwnerReference[]
  resourceVersion: string
  uid: string
  labels: Record<string, string>
}

export interface OwnerReference {
  apiVersion: string
  kind: string
  name: string
  uid: string
}

export interface ESXIMigrationSpec {
  esxiName: string
  openstackCredsRef: NameReference
  rollingMigrationPlanRef: NameReference
  vmwareCredsRef: NameReference
}

export interface NameReference {
  name: string
}

export interface GetESXIMigrationsListMetadata {
  continue: string
  resourceVersion: string
}

export interface ESXIMigrationStatus {
  vms: string[]
  message: string
  phase: string
}

export interface ESXHost {
  id: string
  name: string
  ip: string
  vms: string[]
  state: string
  statusMessage: string
}
