import { ESXHost, ESXIMigration } from './model'

export const getESXiName = (esxiMigration: ESXIMigration): string => {
  return esxiMigration?.spec?.esxiName || ''
}

export const getOpenstackCredsRefName = (esxiMigration: ESXIMigration): string => {
  return esxiMigration?.spec?.openstackCredsRef?.name || ''
}

export const getVMwareCredsRefName = (esxiMigration: ESXIMigration): string => {
  return esxiMigration?.spec?.vmwareCredsRef?.name || ''
}

export const getRollingMigrationPlanRefName = (esxiMigration: ESXIMigration): string => {
  return esxiMigration?.spec?.rollingMigrationPlanRef?.name || ''
}

export const getESXHosts = (esxiMigrations: ESXIMigration[]): ESXHost[] => {
  return esxiMigrations.map((esxiMigration) => ({
    id: esxiMigration.metadata.name,
    name: esxiMigration.spec.esxiName,
    ip: esxiMigration.metadata?.labels?.['vjailbreak.k8s.pf9.io/ip'] || '',
    vms: esxiMigration.status?.vms || [],
    state: esxiMigration.status?.phase || 'Unknown',
    statusMessage: esxiMigration.status?.message || '',
    creationTimestamp: esxiMigration.metadata?.creationTimestamp || ''
  }))
}
