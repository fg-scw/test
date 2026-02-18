// Simplified event system - more flexible and extensible
export const AMPLITUDE_EVENTS = {
  // Migration Events
  MIGRATION_CREATED: 'Migration Created',
  MIGRATION_FAILED: 'Migration Failed', // For creation failures
  MIGRATION_CREATION_FAILED: 'Migration Creation Failed',
  MIGRATION_EXECUTION_FAILED: 'Migration Execution Failed', // For runtime failures
  MIGRATION_SUCCEEDED: 'Migration Succeeded', // For successful completions

  // Credential Events
  CREDENTIALS_ADDED: 'Credentials Added',
  CREDENTIALS_FAILED: 'Credentials Failed',
  ROLLING_MIGRATION_CREATED: 'Rolling Migration Created',
  ROLLING_MIGRATION_SUBMISSION_FAILED: 'Rolling Migration Submission Failed',

  // Cluster Migration Events
  CLUSTER_CONVERSION_TRIGGERED: 'Cluster Conversion Triggered',
  CLUSTER_CONVERSION_FAILED: 'Cluster Conversion Failed',
  CLUSTER_CONVERSION_EXECUTION_FAILED: 'Cluster Conversion Execution Failed', // For runtime failures
  CLUSTER_CONVERSION_SUCCEEDED: 'Cluster Conversion Succeeded' // For successful completions
} as const

export type AmplitudeEventName = (typeof AMPLITUDE_EVENTS)[keyof typeof AMPLITUDE_EVENTS] | string

// Minimal, flexible event properties - only truly universal properties
export interface EventProperties {
  // Universal context properties (always useful)
  component?: string
  userId?: string
  userEmail?: string
  namespace?: string

  // Universal error properties
  errorMessage?: string
  errorCode?: string

  // Flexible property bag - allows any key-value pairs
  [key: string]: unknown
}

// User context for tracking
export interface UserContext {
  userId?: string
  userEmail?: string
  userProperties?: Record<string, unknown>
}
