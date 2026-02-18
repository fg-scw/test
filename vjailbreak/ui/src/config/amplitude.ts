import { EventProperties } from '../types/amplitude'
import { AnalyticsConfig } from '../services/configService'

// Pure Amplitude configuration
export interface AmplitudeConfig {
  apiKey: string
  disabled?: boolean
  releaseStage?: string
  trackingOptions?: {
    ipAddress?: boolean
    deviceId?: boolean
    platform?: boolean
  }
}

// Tracking behavior configuration
export interface TrackingBehavior {
  enabled: boolean
  defaultComponent?: string
  logToConsole?: boolean
}

// Create Amplitude config from ConfigMap or environment fallback
export const createAmplitudeConfig = (configMapData?: AnalyticsConfig): AmplitudeConfig => {
  const isDevelopment = import.meta.env.MODE === 'development'

  // Use ConfigMap data if available, otherwise fall back to environment variables
  const apiKey = configMapData?.amplitude?.apiKey || import.meta.env.VITE_AMPLITUDE_API_KEY || ''
  const releaseStage = configMapData?.releaseStage || import.meta.env.MODE || 'development'
  const hasApiKey = Boolean(apiKey)

  return {
    apiKey,
    releaseStage,
    disabled: isDevelopment && !hasApiKey,
    trackingOptions: {
      ipAddress: true,
      deviceId: true,
      platform: true
    }
  }
}

// Global tracking behavior (simplified)
let trackingBehavior: TrackingBehavior = {
  enabled: true,
  logToConsole: import.meta.env.MODE === 'development'
}

// Functional configuration management
export const getTrackingBehavior = (): TrackingBehavior => trackingBehavior

export const updateTrackingBehavior = (updates: Partial<TrackingBehavior>): void => {
  trackingBehavior = { ...trackingBehavior, ...updates }
}

// Simplified control functions
export const enableTracking = () => updateTrackingBehavior({ enabled: true })
export const disableTracking = () => updateTrackingBehavior({ enabled: false })

// Helper to enrich properties with defaults
export const enrichEventProperties = (
  properties: EventProperties = {},
  defaults: Partial<EventProperties> = {}
): EventProperties => {
  const merged = { ...defaults, ...properties }
  // Filter out undefined values
  return Object.fromEntries(Object.entries(merged).filter(([, value]) => value !== undefined))
}
