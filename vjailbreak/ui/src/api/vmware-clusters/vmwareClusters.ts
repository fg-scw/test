import { VMwareClusterList, VMwareCluster } from './model'
import axios from '../axios'
import { VJAILBREAK_API_BASE_PATH, VJAILBREAK_DEFAULT_NAMESPACE } from '../constants'

export const getVMwareClusters = async (
  namespace = VJAILBREAK_DEFAULT_NAMESPACE,
  vmwareCredName?: string
): Promise<VMwareClusterList> => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/vmwareclusters`

  const config = vmwareCredName
    ? {
        params: {
          labelSelector: `vjailbreak.k8s.pf9.io/vmwarecreds=${vmwareCredName}`
        }
      }
    : undefined

  return axios.get<VMwareClusterList>({
    endpoint,
    config
  })
}

export const getVMwareCluster = async (
  clusterName: string,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
): Promise<VMwareCluster> => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/vmwareclusters/${clusterName}`

  return axios.get<VMwareCluster>({
    endpoint
  })
}
