export interface VersionConfigMap {
  apiVersion: string
  kind: string
  metadata: {
    name: string
    namespace: string
    annotations?: Record<string, string>
  }
  data: {
    upgradeAvailable: string
    upgradeVersion: string
    version: string
  }
}

export interface VersionInfo {
  version: string
  upgradeAvailable: boolean
  upgradeVersion: string
}

export interface ReleaseInfo {
  version: string
  releaseNotes: string
}

export interface AvailableUpdates {
  updates: ReleaseInfo[]
}

export interface ValidationResult {
  agentsScaledDown: boolean
  vmwareCredsDeleted: boolean
  openstackCredsDeleted: boolean
  noMigrationPlans: boolean
  noRollingMigrationPlans: boolean
  noCustomResources: boolean
  crdsCompatible: boolean
  passedAll: boolean
}

export interface UpgradeResponse {
  checks: ValidationResult
  upgradeStarted: boolean
  cleanupRequired?: boolean
  customResourceList?: string[]
}

export interface UpgradeProgressResponse {
  currentStep: string
  progress: number
  status: string
  error: string
  startTime: string
  endTime: string
}
