export interface VjailbreakSettings {
  apiVersion: string
  data: {
    CHANGED_BLOCKS_COPY_ITERATION_THRESHOLD: string
    PERIODIC_SYNC_INTERVAL: string
    PERIODIC_SYNC_TIME_UNIT: string
    CLEANUP_VOLUMES_AFTER_CONVERT_FAILURE: string
    CLEANUP_PORTS_AFTER_MIGRATION_FAILURE: string
    DEFAULT_MIGRATION_METHOD: string
    DEPLOYMENT_NAME: string
    PROXY?: string
    POPULATE_VMWARE_MACHINE_FLAVORS: string
    VCENTER_LOGIN_RETRY_LIMIT: number
    VCENTER_SCAN_CONCURRENCY_LIMIT: number
    VM_ACTIVE_WAIT_INTERVAL_SECONDS: number
    VM_ACTIVE_WAIT_RETRY_LIMIT: number
    AUTO_FSTAB_UPDATE: string
  }
  kind: string
  metadata: {
    annotations?: Record<string, string>
    creationTimestamp: string
    name: string
    namespace: string
    resourceVersion: string
    uid: any
  }
}
