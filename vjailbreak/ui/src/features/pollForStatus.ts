interface Resource {
  kind: string
  metadata: {
    name: string
  }
}

interface PollForStatusParams<T extends Resource> {
  resource: T
  getResourceFunc: (name: string) => Promise<T>
  onUpdate: (resource: T) => void // Called every time the resource is fetched
  stopPollingCond: (resource: T) => boolean
  onSuccess?: (resource: T) => void
  onError?: (error: string) => void // Optional error callback
  pollingInterval?: number
}

export const pollForStatus = <T extends Resource>({
  resource,
  getResourceFunc,
  onUpdate,
  stopPollingCond,
  onSuccess,
  onError,
  pollingInterval = 5000 // Default polling interval to 5 seconds
}: PollForStatusParams<T>) => {
  if (!resource?.metadata?.name) return

  // Timeout after 30 seconds
  const timeoutDuration = 30000

  // Set up timeout to stop polling after 30 seconds
  const timeoutId = setTimeout(() => {
    clearInterval(intervalId) // Stop polling
    if (onError) {
      // onError(`Polling timed out after ${timeoutDuration / 1000} seconds`)
      onError('The request timed out. Please try again.')
    }
  }, timeoutDuration)

  // Start polling
  const intervalId = setInterval(async () => {
    try {
      const updatedResource = await getResourceFunc(resource.metadata.name)
      onUpdate(updatedResource)

      // Check if polling should stop based on the condition
      if (stopPollingCond(updatedResource)) {
        clearInterval(intervalId) // Stop the polling
        clearTimeout(timeoutId) // Clear the timeout if polling finishes early
        if (onSuccess) onSuccess(updatedResource) // Call success callback if provided
      }
    } catch {
      clearInterval(intervalId) // Stop polling in case of error
      clearTimeout(timeoutId) // Clear the timeout
      if (onError) {
        onError(`Error fetching ${resource.kind}`)
      }
    }
  }, pollingInterval)

  // Cleanup function to stop polling manually if needed
  return () => {
    clearInterval(intervalId) // Stop polling
    clearTimeout(timeoutId) // Clear the timeout
  }
}
