import axios from '../axios'
import { VJAILBREAK_API_BASE_PATH, VJAILBREAK_DEFAULT_NAMESPACE } from '../constants'
import { GetClusterMigrationsList, ClusterMigration } from './model'

export const getClusterMigrations = async (
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
): Promise<ClusterMigration[]> => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/clustermigrations`
  const data = await axios.get<GetClusterMigrationsList>({
    endpoint
  })
  return data?.items
}

export const getClusterMigration = async (
  clusterMigrationName: string,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/clustermigrations/${clusterMigrationName}`
  const response = await axios.get<ClusterMigration>({
    endpoint
  })
  return response
}

export const deleteClusterMigration = async (
  clusterMigrationName: string,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/clustermigrations/${clusterMigrationName}`
  const response = await axios.del<ClusterMigration>({
    endpoint
  })
  return response
}
