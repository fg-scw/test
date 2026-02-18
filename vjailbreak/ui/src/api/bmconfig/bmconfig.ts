import axios from '../axios'
import axiosOriginal from 'axios'
import { VJAILBREAK_API_BASE_PATH, VJAILBREAK_DEFAULT_NAMESPACE } from '../constants'
import { GetBMConfigList, BMConfig } from './model'

interface ApiError {
  response?: {
    status?: number
  }
  message: string
}

export const getBMConfigList = async (namespace = VJAILBREAK_DEFAULT_NAMESPACE) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/bmconfigs`
  const response = await axios.get<GetBMConfigList>({
    endpoint
  })
  return response?.items
}

export const getBMConfig = async (bmconfigName, namespace = VJAILBREAK_DEFAULT_NAMESPACE) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/bmconfigs/${bmconfigName}`
  const response = await axios.get<BMConfig>({
    endpoint
  })
  return response
}

export const postBMConfig = async (body, namespace = VJAILBREAK_DEFAULT_NAMESPACE) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/bmconfigs`
  const response = await axios.post<BMConfig>({
    endpoint,
    data: body
  })
  return response
}

export const deleteBMConfig = async (bmconfigName, namespace = VJAILBREAK_DEFAULT_NAMESPACE) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/bmconfigs/${bmconfigName}`
  const response = await axios.del<BMConfig>({
    endpoint
  })
  return response
}

// Create BMConfig with user-data secret reference
export const createBMConfigWithSecret = async (
  configName: string,
  providerType: string,
  apiUrl: string,
  apiKey: string,
  userDataSecretName: string,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE,
  insecure = true,
  os?: string
) => {
  const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/bmconfigs`

  const bmConfigBody = {
    apiVersion: 'vjailbreak.k8s.pf9.io/v1alpha1',
    kind: 'BMConfig',
    metadata: {
      name: configName,
      namespace,
      labels: {
        'app.kubernetes.io/name': 'migration',
        'app.kubernetes.io/part-of': 'vjailbreak'
      }
    },
    spec: {
      providerType,
      apiUrl,
      apiKey,
      userDataSecretRef: {
        name: userDataSecretName,
        namespace
      },
      insecure,
      ...(os ? { os } : {})
    }
  }

  const response = await axios.post<BMConfig>({
    endpoint,
    data: bmConfigBody
  })

  return response
}

export const checkBMConfigExists = async (
  bmconfigName,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE
) => {
  try {
    const endpoint = `${VJAILBREAK_API_BASE_PATH}/namespaces/${namespace}/bmconfigs/${bmconfigName}`
    const response = await axios.get<BMConfig>({
      endpoint
    })
    return { exists: true, config: response }
  } catch (error: unknown) {
    const apiError = error as ApiError
    if (apiError.response && apiError.response.status === 404) {
      return { exists: false, config: null }
    }
    throw error
  }
}

// Interface for boot source data
export interface BootSourceSelection {
  OS: string
  Release: string
  ResourceURI: string
  Arches: string[]
  Subarches: string[]
  Labels: string[]
  ID: number
  BootSourceID: number
}

export interface BootSourceResponse {
  bootSourceSelections: BootSourceSelection[]
}

// Fetch boot sources from MAAS
export const fetchBootSources = async (maasUrl: string, apiKey: string, insecure: boolean) => {
  try {
    const apiUrl = `/dev-api/sdk/vpw/v1/list_boot_source?accessInfo.apiKey=${encodeURIComponent(
      apiKey
    )}&accessInfo.baseUrl=${encodeURIComponent(
      maasUrl
    )}&accessInfo.useInsecure=${insecure}&accessInfo.maas=maas`

    const response = await axiosOriginal.get(apiUrl, {
      headers: {
        accept: 'application/json'
      }
    })

    return response.data
  } catch (error) {
    console.error('Error in fetchBootSources:', error)
    throw error
  }
}
