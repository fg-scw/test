import { Pod, PodListResponse } from './model'
import { KUBERNETES_API_BASE_PATH } from '../constants'
import axios from '../axios'

/**
 * Fetch pods from a namespace with optional label selector
 * @param namespace - The namespace to fetch pods from
 * @param labelSelector - Optional label selector to filter pods
 */
export const fetchPods = async (namespace: string, labelSelector?: string): Promise<Pod[]> => {
  const endpoint = `${KUBERNETES_API_BASE_PATH}/namespaces/${namespace}/pods`

  const config = labelSelector
    ? {
        params: {
          labelSelector
        }
      }
    : undefined

  const response = await axios.get<PodListResponse>({
    endpoint,
    config
  })

  return response.items
}

/**
 * Stream logs from a specific pod
 * @param namespace - The namespace of the pod
 * @param podName - The name of the pod to stream logs from
 * @param options - Optional streaming configuration
 */
export const streamPodLogs = async (
  namespace: string,
  podName: string,
  options: {
    follow?: boolean
    tailLines?: string
    limitBytes?: number
    signal?: AbortSignal
  } = {}
): Promise<Response> => {
  const { follow = true, tailLines = '100', limitBytes = 500000, signal } = options

  const endpoint = `${KUBERNETES_API_BASE_PATH}/namespaces/${namespace}/pods/${podName}/log`

  const params = new URLSearchParams({
    follow: follow.toString(),
    tailLines,
    limitBytes: limitBytes.toString()
  })

  const url = `${endpoint}?${params.toString()}`

  // For streaming endpoints, we need to use raw fetch instead of axios
  const authToken = import.meta.env.VITE_API_TOKEN
  const headers = {
    'Content-Type': 'application/json;charset=UTF-8',
    ...(authToken && { Authorization: `Bearer ${authToken}` })
  }

  const baseUrl = import.meta.env.MODE === 'development' ? '/dev-api' : ''
  const fullUrl = `${baseUrl}${url}`

  const response = await fetch(fullUrl, {
    headers,
    signal
  })

  if (!response.ok) {
    throw new Error(
      `Failed to stream logs from pod ${podName}: HTTP ${response.status}: ${response.statusText}`
    )
  }

  return response
}

/**
 * Fetch debug logs from the mounted /var/log/pf9 directory
 * The UI pod has /var/log/pf9 mounted from the host, and nginx serves these files
 * 
 * Structure:
 * - /var/log/pf9/migration-{name}.log (root level log)
 * - /var/log/pf9/migration-{name}/migration.{timestamp}.log (detailed logs in subdirectory)
 * 
 * @param namespace - The namespace (not used, kept for compatibility)
 * @param podName - The name of the pod (not used, kept for compatibility)
 * @param migrationName - Optional migration name to filter logs
 */
export const fetchPodDebugLogs = async (
  _namespace: string,
  _podName: string,
  migrationName?: string
): Promise<string> => {
  try {
    let combinedLogs = ''
    
    // Helper function to fetch a single file
    const fetchFile = async (path: string, displayName: string): Promise<string> => {
      try {
        const response = await fetch(path)
        if (response.ok) {
          const content = await response.text()
          let result = `\n${'='.repeat(80)}\n`
          result += `FILE: ${displayName}\n`
          result += `${'='.repeat(80)}\n`
          result += content
          result += '\n'
          return result
        } else {
          console.warn(
            `Failed to fetch ${displayName}: HTTP ${response.status}: ${response.statusText} (${path})`
          )
        }
      } catch (error) {
        console.warn(`Failed to fetch ${displayName}:`, error)
      }
      return ''
    }

    // Get the list of items in /var/log/pf9
    const listUrl = '/debug-logs/'
    const listResponse = await fetch(listUrl)
    
    if (!listResponse.ok) {
      console.warn(`Failed to list debug logs: HTTP ${listResponse.status}`)
      return ''
    }

    const rootList = await listResponse.json()
    
    if (!Array.isArray(rootList)) {
      console.warn('Unexpected response format from debug-logs')
      return ''
    }

    // Process each item in the root directory
    for (const item of rootList) {
      if (!item || typeof item !== 'object') {
        console.warn('Skipping unexpected debug-logs root item (not an object):', item)
        continue
      }

      const maybeName = (item as { name?: unknown }).name
      const maybeType = (item as { type?: unknown }).type

      if (typeof maybeName !== 'string' || typeof maybeType !== 'string') {
        console.warn('Skipping unexpected debug-logs root item (missing name/type):', item)
        continue
      }

      const itemName = maybeName
      const itemType = maybeType
      
      // Skip if migration name filter is set and doesn't match
      if (migrationName && !itemName.includes(migrationName)) {
        continue
      }

      if (itemType === 'file' && itemName.endsWith('.log')) {
        // Fetch root-level log file (e.g., migration-{name}.log)
        const content = await fetchFile(`/debug-logs/${itemName}`, itemName)
        combinedLogs += content
      } else if (itemType === 'directory' && itemName.startsWith('migration-')) {
        // Fetch logs from subdirectory (e.g., migration-{name}/migration.{timestamp}.log)
        try {
          const subDirUrl = `/debug-logs/${itemName}/`
          const subDirResponse = await fetch(subDirUrl)
          
          if (subDirResponse.ok) {
            const subDirList = await subDirResponse.json()
            
            if (Array.isArray(subDirList)) {
              for (const subItem of subDirList) {
                if (!subItem || typeof subItem !== 'object') {
                  console.warn(
                    `Skipping unexpected debug-logs subdir item in ${itemName} (not an object):`,
                    subItem
                  )
                  continue
                }

                const subMaybeName = (subItem as { name?: unknown }).name
                const subMaybeType = (subItem as { type?: unknown }).type

                if (typeof subMaybeName !== 'string' || typeof subMaybeType !== 'string') {
                  console.warn(
                    `Skipping unexpected debug-logs subdir item in ${itemName} (missing name/type):`,
                    subItem
                  )
                  continue
                }

                if (subMaybeType === 'file' && subMaybeName.endsWith('.log')) {
                  const content = await fetchFile(
                    `/debug-logs/${itemName}/${subMaybeName}`,
                    `${itemName}/${subMaybeName}`
                  )
                  combinedLogs += content
                }
              }
            }
          } else {
            console.warn(
              `Failed to list debug logs in ${itemName}: HTTP ${subDirResponse.status}: ${subDirResponse.statusText}`
            )
          }
        } catch (error) {
          console.warn(`Failed to fetch logs from subdirectory ${itemName}:`, error)
        }
      }
    }

    if (combinedLogs) {
      console.log(`Fetched ${combinedLogs.length} characters of debug logs`)
    } else {
      console.log('No debug log files found')
    }
    
    return combinedLogs
  } catch (error) {
    console.error('Error fetching debug logs:', error)
    return ''
  }
}
