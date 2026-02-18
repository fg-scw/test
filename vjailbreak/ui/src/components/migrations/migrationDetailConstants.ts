export type MigrationEnvironmentFieldKey =
  | 'sourceDatacenter'
  | 'sourceCluster'
  | 'esxiHost'
  | 'destinationTenant'
  | 'destinationCluster'

export const MIGRATION_ENVIRONMENT_FIELDS: Array<{ key: MigrationEnvironmentFieldKey; label: string }> = [
  { key: 'sourceDatacenter', label: 'Datacenter' },
  { key: 'sourceCluster', label: 'Source Cluster' },
  { key: 'esxiHost', label: 'ESX Host' },
  { key: 'destinationTenant', label: 'Destination Tenant' },
  { key: 'destinationCluster', label: 'Destination Cluster' }
]

export type MigrationPolicyFieldKey =
  | 'securityGroups'
  | 'serverGroup'
  | 'scheduleDataCopy'
  | 'cutoverPolicy'
  | 'renameSuffix'
  | 'folderName'
  | 'disconnectSourceNetwork'
  | 'fallbackToDhcp'
  | 'networkPersistence'
  | 'useGPUFlavor'
  | 'useFlavorless'

export const MIGRATION_POLICY_FIELDS: Array<{ key: MigrationPolicyFieldKey; label: string }> = [
  { key: 'securityGroups', label: 'Security Groups' },
  { key: 'serverGroup', label: 'Server Group' },
  { key: 'scheduleDataCopy', label: 'Schedule Data Copy' },
  { key: 'cutoverPolicy', label: 'Cutover Policy' },
  { key: 'renameSuffix', label: 'Rename Suffix' },
  { key: 'folderName', label: 'Folder Name' },
  { key: 'disconnectSourceNetwork', label: 'Disconnect source network' },
  { key: 'fallbackToDhcp', label: 'Fallback to DHCP' },
  { key: 'networkPersistence', label: 'Persist source network' },
  { key: 'useGPUFlavor', label: 'Use GPU-enabled flavours' },
  { key: 'useFlavorless', label: 'Use dynamic hotplug-enabled flavors' }
]
