import { UseQueryOptions, useQuery } from '@tanstack/react-query'
import { getClusterMigrations } from 'src/api/clustermigrations/clustermigrations'
import { ClusterMigration } from 'src/api/clustermigrations/model'

export const CLUSTER_MIGRATIONS_QUERY_KEY = ['clustermigrations']

export const useClusterMigrationsQuery = (options?: UseQueryOptions<ClusterMigration[], Error>) => {
  return useQuery<ClusterMigration[], Error>({
    queryKey: CLUSTER_MIGRATIONS_QUERY_KEY,
    queryFn: () => getClusterMigrations(),
    ...options
  })
}
