import { useQuery, UseQueryOptions, UseQueryResult } from '@tanstack/react-query'
import { getBMConfig, getBMConfigList } from '../../api/bmconfig/bmconfig'
import { BMConfig } from '../../api/bmconfig/model'
import { VJAILBREAK_DEFAULT_NAMESPACE } from '../../api/constants'

export const BMCONFIG_QUERY_KEY = ['bmconfigs']

type ListOptions = Omit<UseQueryOptions<BMConfig[]>, 'queryKey' | 'queryFn'>
type DetailOptions = Omit<UseQueryOptions<BMConfig>, 'queryKey' | 'queryFn'>

/**
 * Hook to fetch list of BMConfigs
 */
export const useBMConfigsQuery = (
  namespace = VJAILBREAK_DEFAULT_NAMESPACE,
  options: ListOptions = {}
): UseQueryResult<BMConfig[]> => {
  return useQuery<BMConfig[]>({
    queryKey: [...BMCONFIG_QUERY_KEY, namespace],
    queryFn: async () => getBMConfigList(namespace),
    staleTime: Infinity,
    refetchOnWindowFocus: true,
    ...options
  })
}

/**
 * Hook to fetch a single BMConfig by name
 */
export const useBMConfigQuery = (
  name: string,
  namespace = VJAILBREAK_DEFAULT_NAMESPACE,
  options: DetailOptions = {}
): UseQueryResult<BMConfig> => {
  return useQuery<BMConfig>({
    queryKey: [...BMCONFIG_QUERY_KEY, 'detail', name, namespace],
    queryFn: async () => getBMConfig(name, namespace),
    staleTime: Infinity,
    refetchOnWindowFocus: true,
    enabled: !!name,
    ...options
  })
}
