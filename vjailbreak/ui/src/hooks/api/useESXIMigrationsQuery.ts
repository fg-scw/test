import { UseQueryOptions, useQuery } from '@tanstack/react-query'
import { getESXIMigrations } from 'src/api/esximigrations/esximigrations'
import { ESXIMigration } from 'src/api/esximigrations/model'

export const ESXI_MIGRATIONS_QUERY_KEY = ['esximigrations']

export const useESXIMigrationsQuery = (options?: UseQueryOptions<ESXIMigration[], Error>) => {
  return useQuery<ESXIMigration[], Error>({
    queryKey: ESXI_MIGRATIONS_QUERY_KEY,
    queryFn: () => getESXIMigrations(),
    ...options
  })
}
