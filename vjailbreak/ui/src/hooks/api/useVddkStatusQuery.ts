import { useQuery, UseQueryOptions, UseQueryResult } from '@tanstack/react-query'
import { getVddkStatus, VddkStatusResponse } from 'src/api/vddk'

export const VDDK_STATUS_QUERY_KEY = ['vddk-status']

type Options = Omit<UseQueryOptions<VddkStatusResponse>, 'queryKey' | 'queryFn'>

export const useVddkStatusQuery = (options: Options = {}): UseQueryResult<VddkStatusResponse> => {
  return useQuery<VddkStatusResponse>({
    queryKey: VDDK_STATUS_QUERY_KEY,
    queryFn: async () => getVddkStatus(),
    staleTime: 0,
    refetchOnWindowFocus: true,
    retry: 1,
    ...options
  })
}
