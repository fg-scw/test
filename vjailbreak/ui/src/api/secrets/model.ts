import { SecretData } from './secrets'

export interface Secret {
  apiVersion: string
  kind: 'Secret'
  metadata: Metadata
  data: SecretData
  type: 'Opaque'
}

export interface Metadata {
  name: string
  namespace: string
}
