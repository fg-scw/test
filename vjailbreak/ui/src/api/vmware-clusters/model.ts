export interface VMwareCluster {
  apiVersion: string
  kind: string
  metadata: {
    name: string
    namespace: string
    creationTimestamp: string
    generation: number
    resourceVersion: string
    uid: string
    labels?: {
      [key: string]: string
    }
  }
  spec: {
    name: string
    hosts: string[]
  }
}

export interface VMwareClusterList {
  apiVersion: string
  kind: string
  metadata: {
    continue: string
    resourceVersion: string
  }
  items: VMwareCluster[]
}
