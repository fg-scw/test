export interface GetNetworkMappingsList {
  apiVersion: string
  items: NetworkMapping[]
  kind: string
  metadata: GetNetworkMappingsListMetadata
}

export interface NetworkMapping {
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
  networks: Network[]
}

export interface Network {
  source: string
  target: string
}

export interface Status {
  networkMappingValidationMessage: string
  networkMappingValidationStatus: string
}

export interface GetNetworkMappingsListMetadata {
  continue: string
  resourceVersion: string
}
