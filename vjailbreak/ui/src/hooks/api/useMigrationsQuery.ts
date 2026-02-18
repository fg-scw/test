import { useQuery, UseQueryOptions, UseQueryResult } from '@tanstack/react-query'
import { getMigrations } from 'src/api/migrations/migrations'
import { Migration } from 'src/api/migrations/model'

export const MIGRATIONS_QUERY_KEY = ['migrations']

type Options = Omit<UseQueryOptions<Migration[]>, 'queryKey' | 'queryFn'>

export const useMigrationsQuery = (
  namespace = undefined,
  options: Options = {}
): UseQueryResult<Migration[]> => {
  return useQuery<Migration[]>({
    queryKey: [...MIGRATIONS_QUERY_KEY, namespace],
    queryFn: async () => getMigrations(namespace),
    staleTime: Infinity,
    refetchOnWindowFocus: true,
    ...options // Override with custom options
  })
}
