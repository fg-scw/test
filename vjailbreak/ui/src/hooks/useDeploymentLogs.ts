import { useCallback, useEffect, useRef, useState } from 'react'
import { fetchPods, streamPodLogs } from '../api/kubernetes/pods'
import { type Pod } from '../api/kubernetes/model'

interface UseDeploymentLogsParams {
  deploymentName: string
  namespace: string
  labelSelector: string
  enabled: boolean
  sessionKey: number
}

interface UseDeploymentLogsReturn {
  logs: string[]
  isLoading: boolean
  error: string | null
  reconnect: () => void
}

const MAX_LOG_LINES = 5000

export const useDeploymentLogs = ({
  deploymentName,
  namespace,
  labelSelector,
  enabled,
  sessionKey
}: UseDeploymentLogsParams): UseDeploymentLogsReturn => {
  const [logs, setLogs] = useState<string[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const abortControllersRef = useRef<AbortController[]>([])
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const hasInitiallyLoadedRef = useRef(false)
  const previousDeploymentRef = useRef<string>('')
  const seenLogsRef = useRef<Set<string>>(new Set())

  const cleanup = useCallback(() => {
    // Abort all active connections
    abortControllersRef.current.forEach((controller) => {
      controller.abort()
    })
    abortControllersRef.current = []

    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
      reconnectTimeoutRef.current = null
    }
  }, [])

  const fetchPodsForDeployment = useCallback(async (): Promise<Pod[]> => {
    return fetchPods(namespace, labelSelector)
  }, [namespace, labelSelector])

  const streamPodLogsWithProcessing = useCallback(
    async (podName: string, podNamespace: string, fetchHistory: boolean): Promise<void> => {
      const abortController = new AbortController()
      abortControllersRef.current.push(abortController)

      const response = await streamPodLogs(podNamespace, podName, {
        follow: true,
        tailLines: fetchHistory ? '2000' : undefined,
        limitBytes: 8 * 1024 * 1024,
        signal: abortController.signal
      })

      const reader = response.body?.getReader()
      if (!reader) {
        throw new Error(`Response body is not readable for pod ${podName}`)
      }

      const decoder = new TextDecoder()
      let buffer = ''

      // Stream logs using loop to avoid recursion
      let done = false
      while (!done) {
        try {
          const { done: isDone, value } = await reader.read()
          if (isDone) {
            done = true
            // Process any remaining buffer content
            if (buffer.trim()) {
              const logLine = buffer.trim()
              if (!seenLogsRef.current.has(logLine)) {
                seenLogsRef.current.add(logLine)
                setLogs((prevLogs) => {
                  const newLogs = [...prevLogs, logLine]
                  if (newLogs.length > MAX_LOG_LINES) {
                    // Remove oldest from seen set when trimming
                    const removed = newLogs.slice(0, newLogs.length - MAX_LOG_LINES)
                    removed.forEach((log) => seenLogsRef.current.delete(log))
                    return newLogs.slice(-MAX_LOG_LINES)
                  }
                  return newLogs
                })
              }
            }
            break
          }
          buffer += decoder.decode(value, { stream: true })
          const lines = buffer.split('\n')
          buffer = lines.pop() || ''
          if (lines.length > 0) {
            const filteredLines = lines.filter((line) => line.trim())
            const uniqueLines = filteredLines.filter((line) => !seenLogsRef.current.has(line))
            if (uniqueLines.length > 0) {
              uniqueLines.forEach((line) => seenLogsRef.current.add(line))
              setLogs((prevLogs) => {
                const newLogs = [...prevLogs, ...uniqueLines]
                if (newLogs.length > MAX_LOG_LINES) {
                  // Remove oldest from seen set when trimming
                  const removed = newLogs.slice(0, newLogs.length - MAX_LOG_LINES)
                  removed.forEach((log) => seenLogsRef.current.delete(log))
                  return newLogs.slice(-MAX_LOG_LINES)
                }
                return newLogs
              })
            }
          }
        } catch (err) {
          if (err instanceof Error && err.name === 'AbortError') {
            break
          }
          throw err
        }
      }
    },
    []
  )

  const connect = useCallback(async () => {
    if (!enabled || !deploymentName || !namespace || !labelSelector) {
      return
    }

    cleanup()
    setIsLoading(true)
    setError(null)

    try {
      // First, fetch the pods for this deployment
      const pods = await fetchPodsForDeployment()

      if (pods.length === 0) {
        throw new Error(
          `No pods found for deployment ${deploymentName} with label selector ${labelSelector}`
        )
      }

      setIsLoading(false)

      // Start streaming logs from all pods
      const shouldFetchHistory = !hasInitiallyLoadedRef.current
      if (shouldFetchHistory) {
        hasInitiallyLoadedRef.current = true
      }

      const streamPromises = pods.map((pod) =>
        streamPodLogsWithProcessing(pod.metadata.name, pod.metadata.namespace, shouldFetchHistory)
      )

      // Handle each stream promise individually to catch errors
      streamPromises.forEach((promise) => {
        promise.catch((err) => {
          if (err instanceof Error && err.name === 'AbortError') {
            // Aborted, ignore
            return
          }
          console.error('Log streaming error:', err)
          setError(err instanceof Error ? err.message : 'Log streaming error')
        })
      })

      // Streams run indefinitely in parallel, handled individually
    } catch (err) {
      setIsLoading(false)
      if (err instanceof Error && err.name === 'AbortError') {
        // Aborted, don't set error
        return
      }

      const errorMessage =
        err instanceof Error
          ? err.message
          : `Failed to connect to deployment ${deploymentName} logs stream`
      setError(errorMessage)
    }
  }, [
    enabled,
    deploymentName,
    namespace,
    labelSelector,
    cleanup,
    fetchPodsForDeployment,
    streamPodLogsWithProcessing
  ])

  const reconnect = useCallback(() => {
    setLogs([])
    hasInitiallyLoadedRef.current = false
    seenLogsRef.current.clear()
    connect()
  }, [connect])

  useEffect(() => {
    setLogs([])
    hasInitiallyLoadedRef.current = false
    seenLogsRef.current.clear()
  }, [sessionKey])

  useEffect(() => {
    const currentDeploymentKey = `${namespace}/${deploymentName}/${labelSelector}`
    if (previousDeploymentRef.current && previousDeploymentRef.current !== currentDeploymentKey) {
      setLogs([])
      hasInitiallyLoadedRef.current = false
      seenLogsRef.current.clear()
    }
    previousDeploymentRef.current = currentDeploymentKey

    if (enabled && deploymentName && namespace && labelSelector) {
      connect()
    } else {
      cleanup()
      setIsLoading(false)
      setError(null)
    }

    return cleanup
  }, [enabled, deploymentName, namespace, labelSelector, sessionKey, connect, cleanup])

  useEffect(() => {
    return cleanup
  }, [cleanup])

  return {
    logs,
    isLoading,
    error,
    reconnect
  }
}
