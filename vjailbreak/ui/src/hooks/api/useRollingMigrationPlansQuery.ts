import { useQuery, UseQueryOptions } from '@tanstack/react-query'
import { getRollingMigrationPlans } from 'src/api/rolling-migration-plans/rollingMigrationPlans'
import { RollingMigrationPlan } from 'src/api/rolling-migration-plans/model'

export const ROLLING_MIGRATION_PLANS_QUERY_KEY = ['rolling-migration-plans']

export const useRollingMigrationPlansQuery = (
  options?: Omit<UseQueryOptions<RollingMigrationPlan[], Error>, 'queryKey' | 'queryFn'>
) => {
  return useQuery<RollingMigrationPlan[], Error>({
    queryKey: ROLLING_MIGRATION_PLANS_QUERY_KEY,
    queryFn: () => getRollingMigrationPlans(),
    ...options
  })
}
