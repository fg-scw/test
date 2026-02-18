import { v4 as uuidv4 } from 'uuid'

export const createMigrationPlanJson = (params) => {
  const {
    name,
    migrationTemplateName,
    retry = false,
    type = 'hot',
    dataCopyStart,
    vmCutoverStart,
    vmCutoverEnd,
    virtualMachines,
    adminInitiatedCutOver,
    postMigrationAction,
    disconnectSourceNetwork = false,
    securityGroups,
    serverGroup,
    fallbackToDHCP = false,
    postMigrationScript,
    periodicSyncInterval,
    periodicSyncEnabled,
    assignedIPsPerVM,
    networkPersistence,
    acknowledgeNetworkConflictRisk
  } = params || {}

  const spec: Record<string, unknown> = {
    migrationTemplate: migrationTemplateName,
    retry,
    migrationStrategy: {
      type,
      dataCopyStart,
      adminInitiatedCutOver,
      vmCutoverStart,
      vmCutoverEnd,
      disconnectSourceNetwork
    },
    virtualMachines: [virtualMachines],
    fallbackToDHCP
  }

  const advancedOptions: Record<string, unknown> = {}
  if (periodicSyncInterval) {
    advancedOptions.periodicSyncInterval = periodicSyncInterval
  }
  if (typeof periodicSyncEnabled === 'boolean') {
    advancedOptions.periodicSyncEnabled = periodicSyncEnabled
  }
  if (typeof networkPersistence === 'boolean') {
    advancedOptions.networkPersistence = networkPersistence
  }
  if (typeof acknowledgeNetworkConflictRisk === 'boolean') {
    advancedOptions.acknowledgeNetworkConflictRisk = acknowledgeNetworkConflictRisk
  }
  if (Object.keys(advancedOptions).length > 0) {
    spec.advancedOptions = advancedOptions
  }

  if (postMigrationScript && postMigrationScript.trim()) {
    spec.firstBootScript = postMigrationScript
  }

  if (postMigrationAction && (postMigrationAction.renameVm || postMigrationAction.moveToFolder)) {
    spec.postMigrationAction = {
      renameVm: postMigrationAction.renameVm || false,
      suffix: postMigrationAction.suffix || '',
      moveToFolder: postMigrationAction.moveToFolder || false,
      folderName: postMigrationAction.folderName || 'vjailbreakedVMs'
    }
  }

  if (securityGroups && securityGroups.length > 0) {
    spec.securityGroups = securityGroups
  }

  if (serverGroup) {
    spec.serverGroup = serverGroup
  }

  if (assignedIPsPerVM && Object.keys(assignedIPsPerVM).length > 0) {
    spec.assignedIPsPerVM = assignedIPsPerVM
  }

  return {
    apiVersion: 'vjailbreak.k8s.pf9.io/v1alpha1',
    kind: 'MigrationPlan',
    metadata: {
      name: name || uuidv4()
    },
    spec
  }
}
