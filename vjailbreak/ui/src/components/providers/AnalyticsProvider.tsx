import React from 'react'
import { useAnalytics } from 'src/hooks/useAnalytics'

interface AnalyticsProviderProps {
  children: React.ReactNode
}

export function AnalyticsProvider({ children }: AnalyticsProviderProps) {
  useAnalytics()

  return <>{children}</>
}
