import axios from '../axios'
import { VJAILBREAK_API_BASE_PATH, VJAILBREAK_DEFAULT_NAMESPACE } from '../constants'
import { GetESXIMigrationsList, ESXIMigration } from './model'

export const getESXIMigrations = async (
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
): Promise<ESXIMigration[]> => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/esximigrations`
  const data = await axios.get<GetESXIMigrationsList>({
    endpoint
  })
  return data?.items
}

export const getESXIMigration = async (
  esxiMigrationName: string,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/esximigrations/${esxiMigrationName}`
  const response = await axios.get<ESXIMigration>({
    endpoint
  })
  return response
}

export const deleteESXIMigration = async (
  esxiMigrationName: string,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/esximigrations/${esxiMigrationName}`
  const response = await axios.del<ESXIMigration>({
    endpoint
  })
  return response
}
