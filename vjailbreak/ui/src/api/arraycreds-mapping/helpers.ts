import { ArrayCredsMappingFormData } from './model'
import { VJAILBREAK_DEFAULT_NAMESPACE } from '../constants'

export const createArrayCredsMappingJson = (data: ArrayCredsMappingFormData) => {
  const timestamp = Date.now()
  const name = `arraycreds-mapping-${timestamp}`

  return {
    apiVersion: 'vjailbreak.k8s.pf9.io/v1alpha1',
    kind: 'ArrayCredsMapping',
    metadata: {
      name,
      namespace: VJAILBREAK_DEFAULT_NAMESPACE,
    },
    spec: {
      mappings: data.mappings,
    },
  }
}
