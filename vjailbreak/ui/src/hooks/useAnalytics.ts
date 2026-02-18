import { useEffect, useState } from 'react'
import Bugsnag from '@bugsnag/js'
import BugsnagPluginReact from '@bugsnag/plugin-react'
import BugsnagPerformance from '@bugsnag/browser-performance'
import { getBugsnagConfig, getBugsnagPerformanceConfig } from '../config/bugsnag'
import { createAmplitudeConfig } from '../config/amplitude'
import { errorReportingService } from '../services/errorReporting'
import { initializeAmplitude } from '../services/amplitudeService'
import { ConfigService, AnalyticsConfig } from '../services/configService'

export function useAnalytics() {
  const [configMapData, setConfigMapData] = useState<AnalyticsConfig | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [analyticsInitialized, setAnalyticsInitialized] = useState(false)

  // Fetch ConfigMap data on mount
  useEffect(() => {
    let isMounted = true

    async function fetchConfig() {
      try {
        console.log('Fetching analytics configuration from ConfigMap...')
        const data = await ConfigService.fetchAnalyticsConfig()

        if (isMounted) {
          if (data) {
            console.log('ConfigMap data received successfully')
            setConfigMapData(data)
          } else {
            console.warn('ConfigMap fetch returned no data')
            setConfigMapData(null)
          }
          setIsLoading(false)
        }
      } catch (error) {
        console.error('Failed to fetch ConfigMap:', error)
        if (isMounted) {
          setConfigMapData(null)
          setIsLoading(false)
        }
      }
    }

    fetchConfig()

    return () => {
      isMounted = false
    }
  }, [])

  // Initialize analytics when configMapData is available (or when loading completes)
  useEffect(() => {
    if (isLoading || analyticsInitialized) {
      return
    }

    console.log(
      'Initializing analytics...',
      configMapData ? 'with ConfigMap data' : 'with environment variables'
    )

    const bugsnagConfig = getBugsnagConfig(configMapData || undefined)
    const bugsnagPerformanceConfig = getBugsnagPerformanceConfig(configMapData || undefined)
    const amplitudeConfig = createAmplitudeConfig(configMapData || undefined)

    // Initialize Bugsnag if API key is available
    if (bugsnagConfig.apiKey) {
      try {
        Bugsnag.start({
          ...bugsnagConfig,
          plugins: [new BugsnagPluginReact()]
        })

        BugsnagPerformance.start(bugsnagPerformanceConfig)
        errorReportingService.initialize(Bugsnag)
        errorReportingService.addMetadata('app', 'name', 'vjailbreak')
        errorReportingService.addMetadata('app', 'component', 'ui')

        console.log('Bugsnag initialized')
      } catch (error) {
        console.error('Failed to initialize Bugsnag:', error)
      }
    }

    // Initialize Amplitude if API key is available
    if (amplitudeConfig.apiKey) {
      try {
        initializeAmplitude(amplitudeConfig)
        console.log('Amplitude initialized')
      } catch (error) {
        console.error('Failed to initialize Amplitude:', error)
      }
    }

    setAnalyticsInitialized(true)
  }, [configMapData, isLoading, analyticsInitialized])

  return {
    configMapData,
    isLoading,
    analyticsInitialized
  }
}
