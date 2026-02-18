import { useRef, useCallback } from 'react'

// Generic status tracker for migrations
export interface StatusTracker<T = string> {
  [resourceName: string]: {
    previousPhase?: T
    lastReportedPhase?: T
  }
}

// Shared utility for auto-cleanup of old trackers
export const useStatusTracker = <T = string>() => {
  const statusTrackerRef = useRef<StatusTracker<T>>({})

  const autoCleanup = useCallback((activeNames: (string | undefined)[]) => {
    const validNames = activeNames.filter(Boolean) as string[]
    const trackerNames = Object.keys(statusTrackerRef.current)
    trackerNames.forEach((name) => {
      if (!validNames.includes(name)) {
        delete statusTrackerRef.current[name]
      }
    })
  }, [])

  return { statusTrackerRef, autoCleanup }
}
