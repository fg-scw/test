import { ClusterMigration } from './model'

export const getClusterMigrationPhase = (clusterMigration: ClusterMigration): string => {
  return clusterMigration?.status?.phase || 'Unknown'
}

export const getClusterMigrationMessage = (clusterMigration: ClusterMigration): string => {
  return clusterMigration?.status?.message || ''
}

export const getCurrentESXi = (clusterMigration: ClusterMigration): string => {
  return clusterMigration?.status?.currentESXi || ''
}

export const getClusterName = (clusterMigration: ClusterMigration): string => {
  return clusterMigration?.spec?.clusterName || ''
}

export const getESXiMigrationSequence = (clusterMigration: ClusterMigration): string[] => {
  return clusterMigration?.spec?.esxiMigrationSequence || []
}
