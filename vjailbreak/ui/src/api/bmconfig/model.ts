export interface GetBMConfigList {
  apiVersion: string
  items: BMConfig[]
  kind: string
  metadata: GetBMConfigListMetadata
}

export interface BMConfig {
  apiVersion: string
  kind: string
  metadata: ItemMetadata
  spec: BMConfigSpec
  status?: BMConfigStatus
}

export interface ItemMetadata {
  annotations?: Annotations
  creationTimestamp: Date
  generation: number
  name: string
  namespace: string
  resourceVersion: string
  uid: string
  labels?: {
    [key: string]: string
  }
}

export interface Annotations {
  'kubectl.kubernetes.io/last-applied-configuration': string
}

export interface GetBMConfigListMetadata {
  continue: string
  resourceVersion: string
}

export interface BMConfigSpec {
  providerType: string
  apiUrl: string
  apiKey: string
  userDataSecretRef: {
    name: string
    namespace: string
  }
  insecure: boolean
  os?: string // OS/Release from the boot source
}

export interface BMConfigStatus {
  validationMessage?: string
  validationStatus?: string
}
