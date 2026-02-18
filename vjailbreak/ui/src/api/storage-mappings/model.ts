export interface GetStorageMappingsList {
  apiVersion: string
  items: StorageMapping[]
  kind: string
  metadata: GetStorageMappingsListMetadata
}

export interface StorageMapping {
  apiVersion: string
  kind: string
  metadata: ItemMetadata
  spec: Spec
  status: Status
}

export interface ItemMetadata {
  annotations: Annotations
  creationTimestamp: Date
  generation: number
  name: string
  namespace: string
  resourceVersion: string
  uid: string
}

export interface Annotations {
  'kubectl.kubernetes.io/last-applied-configuration': string
}

export interface Spec {
  storages: Storage[]
}

export interface Storage {
  source: string
  target: string
}

export interface Status {
  storageMappingValidationMessage: string
  storageMappingValidationStatus: string
}

export interface GetStorageMappingsListMetadata {
  continue: string
  resourceVersion: string
}
