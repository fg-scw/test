import axios from '../axios'
import {
  VJAILBREAK_API_BASE_PATH,
  VJAILBREAK_DEFAULT_NAMESPACE,
} from '../constants'
import { ArrayCredsMapping } from './model'

interface GetArrayCredsMappingsList {
  items: ArrayCredsMapping[]
}

export const getArrayCredsMappingsList = async (
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/arraycredsmappings`
  const response = await axios.get<GetArrayCredsMappingsList>({
    endpoint,
  })
  return response?.items || []
}

export const getArrayCredsMapping = async (
  arrayCredsMappingName: string,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/arraycredsmappings/${arrayCredsMappingName}`
  const response = await axios.get<ArrayCredsMapping>({
    endpoint,
  })
  return response
}

export const postArrayCredsMapping = async (
  body: any,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/arraycredsmappings`
  const response = await axios.post<ArrayCredsMapping>({
    endpoint,
    data: body,
  })
  return response
}

export const deleteArrayCredsMapping = async (
  arrayCredsMappingName: string,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/arraycredsmappings/${arrayCredsMappingName}`
  const response = await axios.del<ArrayCredsMapping>({
    endpoint,
  })
  return response
}
