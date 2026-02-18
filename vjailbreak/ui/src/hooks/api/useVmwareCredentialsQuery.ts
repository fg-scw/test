import { useQuery, UseQueryOptions, UseQueryResult } from '@tanstack/react-query'
import { getVmwareCredentialsList } from 'src/api/vmware-creds/vmwareCreds'
import { VMwareCreds } from 'src/api/vmware-creds/model'

export const VMWARE_CREDS_QUERY_KEY = ['vmwareCreds']

type Options = Omit<UseQueryOptions<VMwareCreds[]>, 'queryKey' | 'queryFn'>

export const useVmwareCredentialsQuery = (
  namespace = undefined,
  options: Options = {}
): UseQueryResult<VMwareCreds[]> => {
  return useQuery<VMwareCreds[]>({
    queryKey: [...VMWARE_CREDS_QUERY_KEY, namespace],
    queryFn: async () => getVmwareCredentialsList(namespace),
    staleTime: Infinity,
    refetchOnWindowFocus: true,
    ...options // Override with custom options
  })
}
