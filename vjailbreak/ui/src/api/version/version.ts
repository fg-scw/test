import { get, post } from '../axios'
import {
  VersionConfigMap,
  VersionInfo,
  AvailableUpdates,
  UpgradeResponse,
  UpgradeProgressResponse
} from './model'

const VERSION_CONFIG_MAP_NAME = 'version-config'
const VERSION_NAMESPACE = 'migration-system'

export const getVersionConfigMap = async (
  namespace: string = VERSION_NAMESPACE
): Promise<VersionConfigMap> => {
  const endpoint = `/api/v1/namespaces/${namespace}/configmaps/${VERSION_CONFIG_MAP_NAME}`
  return get<VersionConfigMap>({
    endpoint,
    config: { mock: false } // Force real API call, not mock
  })
}

export const getVersionInfo = async (
  namespace: string = VERSION_NAMESPACE
): Promise<VersionInfo> => {
  try {
    const configMap = await getVersionConfigMap(namespace)
    return {
      version: configMap.data.version,
      upgradeAvailable: configMap.data.upgradeAvailable === 'true',
      upgradeVersion: configMap.data.upgradeVersion
    }
  } catch (error) {
    console.error('Failed to fetch version info:', error)
    throw error
  }
}

export const initiateUpgrade = async (
  targetVersion: string,
  autoCleanup: boolean
): Promise<UpgradeResponse> => {
  const endpoint = `/dev-api/sdk/vpw/v1/upgrade`
  return post<UpgradeResponse>({
    endpoint,
    data: { targetVersion, autoCleanup }
  })
}

export const getUpgradeProgress = async (): Promise<UpgradeProgressResponse> => {
  const endpoint = `/dev-api/sdk/vpw/v1/upgrade/progress`
  return get<UpgradeProgressResponse>({ endpoint })
}

export const getAvailableTags = async (): Promise<AvailableUpdates> => {
  const endpoint = '/dev-api/sdk/vpw/v1/tags'
  return get<AvailableUpdates>({ endpoint })
}

export async function cleanupApiCall(): Promise<{ success: boolean; message: string }> {
  return await post<{ success: boolean; message: string }>({
    endpoint: '/dev-api/sdk/vpw/v1/cleanup',
    data: {}
  })
}