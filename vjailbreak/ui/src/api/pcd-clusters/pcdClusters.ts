import { PCDClusterList, PCDCluster } from './model'
import axios from '../axios'
import { VJAILBREAK_API_BASE_PATH, VJAILBREAK_DEFAULT_NAMESPACE } from '../constants'

export const getPCDClusters = async (
  namespace = VJAILBREAK_DEFAULT_NAMESPACE,
  openstackCredName?: string
): Promise<PCDClusterList> => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/pcdclusters`

  const config = openstackCredName
    ? {
        params: {
          labelSelector: `vjailbreak.k8s.pf9.io/openstackcreds=${openstackCredName}`
        }
      }
    : undefined

  return axios.get<PCDClusterList>({
    endpoint,
    config
  })
}

export const getPCDCluster = async (
  clusterName: string,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
): Promise<PCDCluster> => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/pcdclusters/${clusterName}`

  return axios.get<PCDCluster>({
    endpoint
  })
}
