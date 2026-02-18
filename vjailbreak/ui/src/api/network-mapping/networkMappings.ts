import axios from '../axios'
import { VJAILBREAK_API_BASE_PATH, VJAILBREAK_DEFAULT_NAMESPACE } from '../constants'
import { GetNetworkMappingsList, NetworkMapping } from './model'

export const getNetworkMappingList = async (namespace = VJAILBREAK_DEFAULT_NAMESPACE) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/networkmappings`
  const response = await axios.get<GetNetworkMappingsList>({
    endpoint
  })
  return response?.items
}

export const getNetworkMapping = async (
  networkMappingName,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/networkmappings/${networkMappingName}`
  const response = await axios.get<NetworkMapping>({
    endpoint
  })
  return response
}

export const postNetworkMapping = async (body, namespace = VJAILBREAK_DEFAULT_NAMESPACE) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/networkmappings`
  const response = await axios.post<NetworkMapping>({
    endpoint,
    data: body
  })
  return response
}

export const deleteNetworkMapping = async (
  networkMappingName,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/networkmappings/${networkMappingName}`
  const response = await axios.del<NetworkMapping>({
    endpoint
  })
  return response
}
