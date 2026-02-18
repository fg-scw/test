import { v4 as uuidv4 } from 'uuid'

export const createNetworkMappingJson = (params) => {
  const { name, namespace = 'migration-system', networkMappings = [] } = params || {}
  return {
    apiVersion: 'vjailbreak.k8s.pf9.io/v1alpha1',
    kind: 'NetworkMapping',
    metadata: {
      name: name || uuidv4(),
      namespace: namespace
    },
    spec: {
      networks: networkMappings
    }
  }
}
