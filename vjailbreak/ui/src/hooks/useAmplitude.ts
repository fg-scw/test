import { useCallback, useMemo } from 'react'
import { trackEvent, setUserContext, resetUserContext } from '../services/amplitudeService'
import { enrichEventProperties } from '../config/amplitude'
import type { AmplitudeEventName, EventProperties, UserContext } from '../types/amplitude'

export interface UseAmplitudeOptions {
  component?: string
  userId?: string
  userEmail?: string
}

export const useAmplitude = (options: UseAmplitudeOptions = {}) => {
  // Create default properties from options
  const defaultProperties = useMemo(
    (): Partial<EventProperties> => ({
      component: options.component,
      userId: options.userId,
      userEmail: options.userEmail
    }),
    [options.component, options.userId, options.userEmail]
  )

  // Simplified track function
  const track = useCallback(
    (eventName: AmplitudeEventName, properties: EventProperties = {}) => {
      const enrichedProperties = enrichEventProperties(properties, defaultProperties)
      trackEvent(eventName, enrichedProperties)
    },
    [defaultProperties]
  )

  // Async operation wrapper
  const trackAsyncOperation = useCallback(
    async <T>(
      operation: () => Promise<T>,
      successEvent: AmplitudeEventName,
      failureEvent: AmplitudeEventName,
      baseProperties: EventProperties = {}
    ): Promise<T> => {
      try {
        const result = await operation()
        track(successEvent, baseProperties)
        return result
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : String(error)
        track(failureEvent, { ...baseProperties, errorMessage })
        throw error
      }
    },
    [track]
  )

  // User management
  const setUser = useCallback((user: UserContext) => {
    setUserContext(user)
  }, [])

  const resetUser = useCallback(() => {
    resetUserContext()
  }, [])

  return {
    track,
    trackAsyncOperation,
    setUser,
    resetUser
  }
}

export default useAmplitude
