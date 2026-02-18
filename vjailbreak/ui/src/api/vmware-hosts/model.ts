export interface VMwareHost {
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
    hardwareUuid: string
    hostConfigId?: string
  }
}

export interface VMwareHostList {
  apiVersion: string
  kind: string
  metadata: {
    continue: string
    resourceVersion: string
  }
  items: VMwareHost[]
}
