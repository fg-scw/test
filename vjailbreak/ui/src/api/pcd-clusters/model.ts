export interface PCDCluster {
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
    clusterName: string
    hosts: string[]
    rebalancingFrequencyMins?: number
  }
  status?: Record<string, unknown>
}

export interface PCDClusterList {
  apiVersion: string
  kind: string
  metadata: {
    continue: string
    resourceVersion: string
  }
  items: PCDCluster[]
}
