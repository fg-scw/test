export interface ArrayCredsMapping {
  apiVersion: string
  kind: string
  metadata: {
    name: string
    namespace: string
    creationTimestamp?: string
    uid?: string
  }
  spec: ArrayCredsMappingSpec
  status?: ArrayCredsMappingStatus
}

export interface ArrayCredsMappingSpec {
  mappings: DatastoreArrayCredsMapping[]
}

export interface DatastoreArrayCredsMapping {
  source: string
  target: string
}

export interface ArrayCredsMappingStatus {
  validationStatus?: string
  validationMessage?: string
}

export interface ArrayCredsMappingFormData {
  mappings: Array<{ source: string; target: string }>
}
