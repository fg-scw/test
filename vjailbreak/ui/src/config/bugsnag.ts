import { AnalyticsConfig } from '../services/configService'

export interface BugsnagConfig {
  apiKey: string
  appVersion: string
  releaseStage: string
  enabledReleaseStages: string[]
  collectUserIp: boolean
  autoDetectErrors: boolean
  autoTrackSessions: boolean
}

export interface BugsnagPerformanceConfig {
  apiKey: string
}

export const getBugsnagConfig = (configMapData?: AnalyticsConfig): BugsnagConfig => {
  const apiKey = configMapData?.bugsnag?.apiKey || import.meta.env.VITE_BUGSNAG_API_KEY || ''

  const appVersion =
    configMapData?.bugsnag?.appVersion || import.meta.env.VITE_APP_VERSION || '1.0.0'

  const releaseStage = configMapData?.releaseStage || import.meta.env.MODE || 'development'

  // Determine enabled release stages based on the actual release stage
  const enabledReleaseStages =
    releaseStage === 'production' ? ['production'] : ['development', 'staging', 'production']

  return {
    apiKey,
    appVersion,
    releaseStage,
    enabledReleaseStages,
    collectUserIp: false,
    autoDetectErrors: true,
    autoTrackSessions: releaseStage !== 'development'
  }
}

export const getBugsnagPerformanceConfig = (
  configMapData?: AnalyticsConfig
): BugsnagPerformanceConfig => {
  const apiKey = configMapData?.bugsnag?.apiKey || import.meta.env.VITE_BUGSNAG_API_KEY || ''

  return {
    apiKey
  }
}
