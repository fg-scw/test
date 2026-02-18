import { useMemo } from 'react'
import { useQuery, UseQueryResult } from '@tanstack/react-query'

import { getMigrationPlan } from 'src/api/migration-plans/migrationPlans'
import { getMigrationTemplate } from 'src/api/migration-templates/migrationTemplates'
import { getOpenstackCredentials } from 'src/api/openstack-creds/openstackCreds'
import type { Migration } from './migrations'

export type MigrationPlanDestination = {
  destinationCluster: string
  destinationTenant: string
}

export type MigrationPlanDestinationsByKey = Record<string, MigrationPlanDestination>

export const useMigrationPlanDestinationsQuery = (
  migrations: Migration[]
): UseQueryResult<MigrationPlanDestinationsByKey> => {
  const planQueryKey = useMemo(() => {
    const namespaces = Array.from(
      new Set(
        migrations
          .map((m) => m.metadata?.namespace)
          .filter(Boolean)
          .map(String)
      )
    )
      .sort()
      .join(',')

    const planNames = Array.from(
      new Set(
        migrations
          .map((m) => (m.spec as any)?.migrationPlan || (m.metadata as any)?.labels?.migrationplan)
          .filter(Boolean)
          .map(String)
      )
    )
      .sort()
      .join(',')

    return ['migration-plan-destinations', namespaces, planNames]
  }, [migrations])

  return useQuery({
    queryKey: planQueryKey,
    enabled: migrations.length > 0,
    refetchOnWindowFocus: false,
    staleTime: 60_000,
    queryFn: async () => {
      const safeGet = async <T,>(fn: () => Promise<T>): Promise<T | null> => {
        try {
          return await fn()
        } catch (error) {
          console.error('Error in safeGet:', error)
          return null
        }
      }

      const planKeys = Array.from(
        new Set(
          migrations
            .map((m) => {
              const namespace = m.metadata?.namespace
              const planName = (m.spec as any)?.migrationPlan || (m.metadata as any)?.labels?.migrationplan
              if (!namespace || !planName) return ''
              return `${namespace}::${planName}`
            })
            .filter(Boolean)
        )
      )

      const results = await Promise.all(
        planKeys.map(async (key) => {
          const [namespace, planName] = key.split('::')
          const plan = await safeGet(() => getMigrationPlan(planName, namespace))
          const templateName = (plan?.spec as any)?.migrationTemplate as string | undefined
          const template = templateName ? await safeGet(() => getMigrationTemplate(templateName, namespace)) : null
          const templateSpec = (template?.spec as any) || {}
          const openstackRef = templateSpec?.destination?.openstackRef as string | undefined
          const creds = openstackRef ? await safeGet(() => getOpenstackCredentials(openstackRef, namespace)) : null

          const destinationCluster = (templateSpec?.targetPCDClusterName as string) || 'N/A'
          const destinationTenant = (creds?.spec as any)?.projectName || 'N/A'
          return [key, { destinationCluster, destinationTenant }] as const
        })
      )

      return Object.fromEntries(results)
    }
  })
}
