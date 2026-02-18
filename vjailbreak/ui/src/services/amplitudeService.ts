import * as amplitude from '@amplitude/analytics-browser'
import type { AmplitudeEventName, EventProperties, UserContext } from '../types/amplitude'
import type { AmplitudeConfig } from '../config/amplitude'
import { getTrackingBehavior } from '../config/amplitude'

// Simple state management
let isInitialized = false

// Initialize Amplitude
export const initializeAmplitude = (config: AmplitudeConfig): boolean => {
  if (config.disabled) {
    console.log('Amplitude disabled in current environment')
    return false
  }

  try {
    const initOptions = {
      useDynamicConfig: true, // Let the SDK fetch the best endpoint
      ...(config.trackingOptions && {
        trackingOptions: config.trackingOptions
      })
    }

    amplitude.init(config.apiKey, undefined, initOptions)

    // Set release stage as a global user property if available
    if (config.releaseStage) {
      const identify = new amplitude.Identify()
      identify.setOnce('release_stage', config.releaseStage)
      amplitude.identify(identify)
    }

    isInitialized = true
    console.log('Amplitude initialized successfully')
    return true
  } catch (error) {
    console.error('Failed to initialize Amplitude:', error)
    return false
  }
}

// Check if ready
export const isAmplitudeReady = (): boolean => isInitialized

// Set user context
export const setUserContext = (user: UserContext): void => {
  if (!isAmplitudeReady()) return

  try {
    if (user.userId) {
      amplitude.setUserId(user.userId)
    }

    if (user.userProperties) {
      const identify = new amplitude.Identify()
      Object.entries(user.userProperties).forEach(([key, value]) => {
        identify.setOnce(key, value as string | number | boolean)
      })
      amplitude.identify(identify)
    }
  } catch (error) {
    console.error('Failed to set user context:', error)
  }
}

// Core tracking function - simplified and more robust
export const trackEvent = (
  eventName: AmplitudeEventName,
  properties: EventProperties = {}
): void => {
  const behavior = getTrackingBehavior()

  // Check if tracking is enabled
  if (!behavior.enabled) {
    if (behavior.logToConsole) {
      console.log('Tracking disabled:', eventName, properties)
    }
    return
  }

  // Check if Amplitude is ready
  if (!isAmplitudeReady()) {
    console.warn('Amplitude not initialized, skipping event:', eventName)
    return
  }

  try {
    // Clean properties (remove undefined values)
    const cleanProperties = Object.fromEntries(
      Object.entries(properties).filter(([, value]) => value !== undefined)
    )

    amplitude.track(eventName, cleanProperties)

    if (behavior.logToConsole) {
      console.log('📊 Amplitude event:', eventName, cleanProperties)
    }
  } catch (error) {
    console.error('Failed to track event:', error)
  }
}

// Reset user data
export const resetUserContext = (): void => {
  if (!isAmplitudeReady()) return

  try {
    amplitude.reset()
  } catch (error) {
    console.error('Failed to reset user context:', error)
  }
}

// Flush pending events
export const flushEvents = async (): Promise<void> => {
  if (!isAmplitudeReady()) return Promise.resolve()

  try {
    await amplitude.flush()
  } catch (error) {
    console.error('Failed to flush events:', error)
  }
}

// Legacy compatibility - can be removed later
export const amplitudeService = {
  initialize: initializeAmplitude,
  isReady: isAmplitudeReady,
  track: trackEvent,
  setUser: setUserContext,
  reset: resetUserContext,
  flush: flushEvents
}

export default amplitudeService
