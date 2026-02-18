export interface GetArrayCredsList {
  apiVersion: string
  items: ArrayCreds[]
  kind: string
  metadata: GetArrayCredsListMetadata
}

export interface ArrayCreds {
  apiVersion: string
  kind: string
  metadata: ArrayCredsMetadata
  spec: ArrayCredsSpec
  status?: ArrayCredsStatus
}

export interface ArrayCredsMetadata {
  annotations?: Annotations
  creationTimestamp?: Date
  generation?: number
  name: string
  namespace: string
  resourceVersion?: string
  uid?: string
  labels?: Record<string, string>
}

export interface Annotations {
  'kubectl.kubernetes.io/last-applied-configuration'?: string
}

export interface DatastoreInfo {
  name: string
  type: string
  capacity: number
  freeSpace: number
  backingNAA: string
  backingUUID: string
  moID: string
}

export interface OpenstackMapping {
  volumeType: string
  cinderBackendName: string
  cinderBackendPool?: string
  cinderHost?: string
}

export interface ArrayCredsSpec {
  vendorType: string
  secretRef?: {
    name: string
  }
  openstackMapping?: OpenstackMapping
  autoDiscovered?: boolean
}

export interface GetArrayCredsListMetadata {
  continue: string
  resourceVersion: string
}

export interface ArrayCredsStatus {
  arrayValidationStatus?: string
  arrayValidationMessage?: string
  dataStore?: DatastoreInfo[]
  phase?: string
}

// Supported vendor types for storage arrays
export const ARRAY_VENDOR_TYPES = [
  { value: 'pure', label: 'Pure Storage' },
  { value: 'netapp', label: 'NetApp Storage' },
  { value: 'unsupported', label: 'N/A' }
] as const

export type ArrayVendorType = (typeof ARRAY_VENDOR_TYPES)[number]['value']
