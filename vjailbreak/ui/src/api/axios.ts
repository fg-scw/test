import axios from 'axios'
import { pathJoin } from 'src/utils'
import { AxiosRequestConfig } from 'axios'

interface ExtendedAxiosConfig extends AxiosRequestConfig {
  mock?: boolean
}

interface BaseRequestParams {
  endpoint: string
  baseUrl?: string
  config?: ExtendedAxiosConfig
}

interface RequestParamsWithData extends BaseRequestParams {
  data: unknown
}

const getHeaders = () => {
  const authToken = import.meta.env.VITE_API_TOKEN
  const headers = {
    common: {
      'Content-Type': 'application/json;charset=UTF-8',
      ...(authToken && { Authorization: `Bearer ${authToken}` })
    }
  }
  return headers
}

const axiosInstance = axios.create({
  headers: getHeaders(),
  withCredentials: true
})

const getDefaultBaseUrl = (config?: ExtendedAxiosConfig) => {
  if (import.meta.env.VITE_USE_MOCK_API === 'true') {
    if (config?.mock === false) {
      return import.meta.env.MODE === 'development' ? '/dev-api' : ''
    }
    return 'http://localhost:3001/mock-server'
  }

  if (import.meta.env.MODE === 'development') {
    return '/dev-api'
  }
  return ''
}

interface AxiosPatchRequestParams {
  endpoint: string
  baseUrl?: string
  data: unknown
  config?: AxiosRequestConfig
}

// Wrappers for axios methods
export const get = async <T>({ endpoint, baseUrl, config }: BaseRequestParams) => {
  const url = pathJoin(baseUrl || getDefaultBaseUrl(config), endpoint)
  const response = await axiosInstance.get<T>(url, config)
  return response.data
}

export const post = async <T>({ endpoint, baseUrl, data, config = {} }: RequestParamsWithData) => {
  const url = pathJoin(baseUrl || getDefaultBaseUrl(config), endpoint)
  const response = await axiosInstance.post<T>(url, data, config)
  return response.data
}

export const put = async <T>({ endpoint, baseUrl, data, config = {} }: RequestParamsWithData) => {
  const url = pathJoin(baseUrl || getDefaultBaseUrl(config), endpoint)
  const response = await axiosInstance.put<T>(url, data, config)
  return response.data
}

const patch = async <T>({ endpoint, baseUrl, data, config = {} }: AxiosPatchRequestParams) => {
  const url = pathJoin(baseUrl || getDefaultBaseUrl(), endpoint)
  const response = await axiosInstance.patch<T>(url, data, config)
  return response.data
}

export const del = async <T>({ endpoint, baseUrl, config = {} }: BaseRequestParams) => {
  const url = pathJoin(baseUrl || getDefaultBaseUrl(config), endpoint)
  const response = await axiosInstance.delete<T>(url, config)
  return response.data
}

export default {
  get,
  post,
  put,
  patch,
  del
}
