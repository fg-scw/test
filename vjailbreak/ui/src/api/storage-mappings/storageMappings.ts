import axios from '../axios'
import { VJAILBREAK_API_BASE_PATH, VJAILBREAK_DEFAULT_NAMESPACE } from '../constants'
import { GetStorageMappingsList, StorageMapping } from './model'

export const getStorageMappingsList = async (namespace = VJAILBREAK_DEFAULT_NAMESPACE) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/storagemappings`
  const response = await axios.get<GetStorageMappingsList>({
    endpoint
  })
  return response?.items
}

export const getStorageMapping = async (
  storageMappingName,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/storagemappings/${storageMappingName}`
  const response = await axios.get<StorageMapping>({
    endpoint
  })
  return response
}

export const postStorageMapping = async (body, namespace = VJAILBREAK_DEFAULT_NAMESPACE) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/storagemappings`
  const response = await axios.post<StorageMapping>({
    endpoint,
    data: body
  })
  return response
}

export const deleteStorageMapping = async (
  storageMappingName,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/storagemappings/${storageMappingName}`
  const response = await axios.del<StorageMapping>({
    endpoint
  })
  return response
}
