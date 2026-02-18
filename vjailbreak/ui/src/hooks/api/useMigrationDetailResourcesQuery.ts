import { useQuery, UseQueryResult } from '@tanstack/react-query'

import type { MigrationPlan } from 'src/api/migration-plans/model'
import { getMigrationPlan } from 'src/api/migration-plans/migrationPlans'
import type { MigrationTemplate } from 'src/api/migration-templates/model'
import { getMigrationTemplate } from 'src/api/migration-templates/migrationTemplates'
import type { NetworkMapping } from 'src/api/network-mapping/model'
import { getNetworkMapping } from 'src/api/network-mapping/networkMappings'
import type { OpenstackCreds } from 'src/api/openstack-creds/model'
import { getOpenstackCredentials } from 'src/api/openstack-creds/openstackCreds'
import type { RdmDisk } from 'src/api/rdm-disks/model'
import { getRdmDisksList } from 'src/api/rdm-disks/rdmDisks'
import type { StorageMapping } from 'src/api/storage-mappings/model'
import { getStorageMapping } from 'src/api/storage-mappings/storageMappings'
import type { VMwareMachine } from 'src/api/vmware-machines/model'
import { getVMwareMachines } from 'src/api/vmware-machines/vmwareMachines'
import type { Migration } from 'src/features/migration/api/migrations'

export interface MigrationDetailResources {
  migrationPlan: MigrationPlan | null
  migrationTemplate: MigrationTemplate | null
  openstackCreds: OpenstackCreds | null
  networkMapping: NetworkMapping | null
  storageMapping: StorageMapping | null
  vmwareMachine: VMwareMachine | null
  rdmDisks: RdmDisk[]
}

export const useMigrationDetailResourcesQuery = ({
  open,
  migration
}: {
  open: boolean
  migration: Migration | null
}): UseQueryResult<MigrationDetailResources, Error> => {
  return useQuery<MigrationDetailResources>({
    queryKey: ['migration-detail', migration?.metadata?.namespace, migration?.metadata?.name],
    enabled: open && Boolean(migration?.metadata?.name),
    refetchOnWindowFocus: false,
    staleTime: 0,
    queryFn: async (): Promise<MigrationDetailResources> => {
      if (!migration) {
        return {
          migrationPlan: null,
          migrationTemplate: null,
          openstackCreds: null,
          networkMapping: null,
          storageMapping: null,
          vmwareMachine: null,
          rdmDisks: []
        }
      }

      const namespace = migration?.metadata?.namespace
      const migrationSpec = migration?.spec as any
      const vmName = (migrationSpec?.vmName as string) || ''
      const vmStableId =
        (migrationSpec?.vmId as string) ||
        (migrationSpec?.vmID as string) ||
        (migrationSpec?.vmwareMachine as string) ||
        (migrationSpec?.vmwareMachineName as string) ||
        (migrationSpec?.vmwareMachineRef as string) ||
        ''
      const migrationPlanName =
        (migrationSpec?.migrationPlan as string) || (migration?.metadata as any)?.labels?.migrationplan

      const safeGet = async <T,>(fn: () => Promise<T>): Promise<T | null> => {
        try {
          return await fn()
        } catch (error) {
          console.error('Error in safeGet:', error)
          return null
        }
      }

      const migrationPlan = migrationPlanName ? await safeGet(() => getMigrationPlan(migrationPlanName, namespace)) : null

      const migrationTemplateName = (migrationPlan?.spec as any)?.migrationTemplate as string | undefined
      const migrationTemplate = migrationTemplateName
        ? await safeGet(() => getMigrationTemplate(migrationTemplateName, namespace))
        : null

      const templateSpec = (migrationTemplate?.spec as any) || {}
      const openstackRef = templateSpec?.destination?.openstackRef as string | undefined
      const networkMappingName = templateSpec?.networkMapping as string | undefined
      const storageMappingName = templateSpec?.storageMapping as string | undefined
      const vmwareRef = templateSpec?.source?.vmwareRef as string | undefined

      const [openstackCreds, networkMapping, storageMapping] = await Promise.all([
        openstackRef ? safeGet(() => getOpenstackCredentials(openstackRef, namespace)) : Promise.resolve(null),
        networkMappingName ? safeGet(() => getNetworkMapping(networkMappingName, namespace)) : Promise.resolve(null),
        storageMappingName ? safeGet(() => getStorageMapping(storageMappingName, namespace)) : Promise.resolve(null)
      ])

      const vmwareMachinesList = await safeGet(() => getVMwareMachines(namespace, vmwareRef))
      const vmwareMachines = vmwareMachinesList?.items || []

      const vmwareMachine =
        vmwareMachines.length
          ? vmwareMachines.find((m) => vmStableId && m?.metadata?.name === vmStableId) ||
            vmwareMachines.find((m) => vmName && (m?.spec as any)?.vms?.name === vmName) ||
            null
          : null

      const rdmDiskNames = ((vmwareMachine?.spec as any)?.vms?.rdmDisks as string[]) || []
      const effectiveVmName = vmName || ((vmwareMachine?.spec as any)?.vms?.name as string) || ''
      const allRdmDisks = effectiveVmName ? await safeGet(() => getRdmDisksList(namespace)) : null
      const rdmDisks = (allRdmDisks || []).filter((d: any) => {
        const ownerVMs = (d?.spec?.ownerVMs as string[]) || []
        const diskName = (d?.spec?.diskName as string) || ''
        const metaName = (d?.metadata?.name as string) || ''
        return (
          (effectiveVmName && ownerVMs.includes(effectiveVmName)) ||
          (rdmDiskNames.length && (rdmDiskNames.includes(metaName) || rdmDiskNames.includes(diskName)))
        )
      })

      return {
        migrationPlan,
        migrationTemplate,
        openstackCreds,
        networkMapping,
        storageMapping,
        vmwareMachine,
        rdmDisks
      }
    }
  })
}
